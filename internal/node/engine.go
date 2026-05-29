// internal/node/engine.go
package node

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
)

// RunResult 单节点执行结果. framework 用来路由 + 日志 + error 传播.
type RunResult struct {
	ExitName    string
	OutputData  map[string]any
	Error       error
	Validation  []ValidationError
	Panic       any    // recovered panic value, nil if no panic
	PanicStack  string // stack trace if panic
	DisplayText string // Display() 返回值, "" = 不 emit
}

// preparedExec is the result of phases 1-3 (gates passed, ctx built). 内部 struct.
type preparedExec struct {
	in  Inputs
	ctx Ctx
}

// prepareExec runs phases 1-3: build inputs, Required gate, Validate gate.
// Gate 失败 → 返 nil + RunResult{Validation: errs}.
func prepareExec(ctx context.Context, rn *RegisteredNode, dataWire, config, execData map[string]any, services ServiceBundle) (*preparedExec, *RunResult) {
	in := newInputs(dataWire, config, execData, rn.Defaults)
	if errs := validateRequired(&rn.Spec, in); len(errs) > 0 {
		return nil, &RunResult{Validation: errs}
	}
	if rn.Validate != nil {
		if errs := rn.Validate(in); len(errs) > 0 {
			return nil, &RunResult{Validation: errs}
		}
	}
	c := newCtx(ctx, services, &rn.Spec)
	return &preparedExec{in: in, ctx: c}, nil
}

// runWithRecover 跑 fn(), 包装 panic recover + Display 联动. fn 是 closure 包
// rn.Run(c, in) / rn.RunRegion(c, in, body). 用 named return 让 deferred recover
// 写 result 后正常返回 (而非 panic 继续传播).
func runWithRecover(rn *RegisteredNode, p *preparedExec, fn func() (Outputs, error)) (result RunResult) {
	defer func() {
		if r := recover(); r != nil {
			result.Panic = r
			result.PanicStack = string(debug.Stack())
			// result.ExitName/OutputData 在 fn() 返回后已赋值, Display callback 可能 panic.
			// 清空以防 dispatch 把 half-baked result 当合法路由.
			result.ExitName = ""
			result.OutputData = nil
			result.DisplayText = ""
		}
	}()
	outs, err := fn()
	if err != nil {
		result.Error = err
		return result
	}
	if outs == nil {
		return result
	}
	name, data := outs.exit()
	result.ExitName = name
	result.OutputData = data
	if rn.Display != nil {
		od := &outputDataImpl{data: data}
		result.DisplayText = rn.Display(p.in, name, od)
	}
	return result
}

// RunNode framework 入口. 调用方传 RegisteredNode + 已 merged inputs sources + services bundle.
//
// 错误分类 (spec v3 §4.2):
//   - Validation:  user graph 写错 (Required 缺值 / Validator 返错) → 节点变红 (NOT panic)
//   - Error:       runtime fail (Run 返 error) → 节点变红
//   - Panic:       framework invariant broken (double Fire / Out(unknown) / impossible state) → recover + stack 进 log
//
// services 内任何字段都可以 nil — 节点拿到 nil service 调方法时 panic, 由 framework
// recover 报回 RunResult.Panic. 测试可用 StubServices() 一键填齐.
func RunNode(ctx context.Context, rn *RegisteredNode, dataWire, config, execData map[string]any, services ServiceBundle) RunResult {
	if rn.Run == nil {
		return RunResult{Error: fmt.Errorf("node %q is not Runnable", rn.Spec.Kind)}
	}
	p, gateFail := prepareExec(ctx, rn, dataWire, config, execData, services)
	if gateFail != nil {
		return *gateFail
	}
	return runWithRecover(rn, p, func() (Outputs, error) {
		return rn.Run(p.ctx, p.in)
	})
}

// RunNodeAsRegion framework 入口供 region runner 调用 RegionRunner-implementing 节点.
// 跟 RunNode 同套 Required/Validate/recover/Display 管线, 区别在调 rn.RunRegion(c, in, body)
// 而非 rn.Run(c, in).
//
// body 是 "执行 region 内部下游" 的回调, 由调用方 (region runner) 提供 — 节点决定
// 调 0 / 1 / N 次. body 返 error → 节点可截获 (Try) / 翻译 sentinel (Loop Break/Continue)
// 或直接 propagate.
//
// 节点没实现 RegionRunner → RunResult.Error = "not a RegionRunner".
func RunNodeAsRegion(ctx context.Context, rn *RegisteredNode, dataWire, config, execData map[string]any, services ServiceBundle, body func(Ctx) error) RunResult {
	if rn.RunRegion == nil {
		return RunResult{Error: fmt.Errorf("node %q is not a RegionRunner", rn.Spec.Kind)}
	}
	p, gateFail := prepareExec(ctx, rn, dataWire, config, execData, services)
	if gateFail != nil {
		return *gateFail
	}
	return runWithRecover(rn, p, func() (Outputs, error) {
		return rn.RunRegion(p.ctx, p.in, body)
	})
}

// EvaluatePureData framework 入口供 pure-data 节点求值 (no exec / no token).
// 走跟 RunNode 相同的 Required + Validate 管线, 但 Run 路径换成 rn.Evaluate.
//
// caller (dispatch_v5) 准备 dataWire (上游已 pull 完) + config; 节点 Evaluate 实现直接 in.Float64(...) 读.
// 没实现 Evaluator → 返 error (caller fallback 老 path).
//
// Required 缺值 / Validate 失败 → 返 error (用 ValidationError.Message 拼).
// Run panic recover, panic 信息塞 error.
func EvaluatePureData(ctx context.Context, rn *RegisteredNode, dataWire, config map[string]any, services ServiceBundle) (any, error) {
	if !rn.Spec.IsPureData {
		return nil, fmt.Errorf("EvaluatePureData: kind %q is not IsPureData", rn.Spec.Kind)
	}
	if rn.Evaluate == nil {
		return nil, fmt.Errorf("EvaluatePureData: kind %q does not implement Evaluator", rn.Spec.Kind)
	}
	// snapshot wrap — PureData Evaluator 内部看到的 Vars/Sys 是 tick-frozen view.
	// services.Snapshot 是 framework-injected getter; nil 时跳过 wrap (测试 stub 用).
	// services 是值类型 (传值 copy), 改 services 内字段不影响 caller.
	if services.Snapshot != nil {
		snap := services.Snapshot(ctx)
		services.Vars = newSnapshotVarStore(services.Vars, snap)
		if snap.Sys != nil {
			services.Sys = snap.Sys
		}
	}
	p, gateFail := prepareExec(ctx, rn, dataWire, config, nil, services)
	if gateFail != nil {
		msgs := make([]string, 0, len(gateFail.Validation))
		for _, e := range gateFail.Validation {
			msgs = append(msgs, e.Message)
		}
		return nil, fmt.Errorf("EvaluatePureData %s: %s", rn.Spec.Kind, strings.Join(msgs, "; "))
	}
	var v any
	var runErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				runErr = fmt.Errorf("EvaluatePureData %s panic: %v\n%s", rn.Spec.Kind, r, debug.Stack())
			}
		}()
		v, runErr = rn.Evaluate(p.ctx, p.in)
	}()
	return v, runErr
}

// outputDataImpl 给 Display 用. snapshot, immutable view of OutputData map.
type outputDataImpl struct{ data map[string]any }

func (o *outputDataImpl) Has(field string) bool { _, ok := o.data[field]; return ok }
func (o *outputDataImpl) Raw(field string) any  { return o.data[field] }

func (o *outputDataImpl) String(field string) string {
	v, _ := o.data[field].(string)
	return v
}

func (o *outputDataImpl) Bool(field string) bool {
	v, _ := o.data[field].(bool)
	return v
}

func (o *outputDataImpl) Int(field string) int {
	switch v := o.data[field].(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}

func (o *outputDataImpl) Float64(field string) float64 {
	switch v := o.data[field].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	}
	return 0
}

func (o *outputDataImpl) Point(field string) Point {
	v, _ := o.data[field].(Point)
	return v
}

func defaultsFromSpec(spec *Spec) map[string]any {
	d := map[string]any{}
	for _, in := range spec.Inputs {
		if in.Default != nil {
			d[in.Name] = in.Default
		}
	}
	return d
}

func validateRequired(spec *Spec, in Inputs) []ValidationError {
	var errs []ValidationError
	for _, input := range spec.Inputs {
		if !input.Required {
			continue
		}
		if input.Type == TypeExec {
			continue // exec pin 不算 data required
		}
		if !in.Has(input.Name) {
			errs = append(errs, ValidationError{
				Code:    "REQUIRED_FIELD_MISSING",
				Message: fmt.Sprintf("required field %q missing", input.Name),
				Field:   input.Name,
			})
		}
	}
	return errs
}
