// internal/node/engine.go
package node

import (
	"context"
	"fmt"
	"runtime/debug"
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

// RunNode framework 入口. 调用方传 RegisteredNode + 已 merged inputs sources.
//
// 错误分类 (spec v3 §4.2):
//   - Validation:  user graph 写错 (Required 缺值 / Validator 返错) → 节点变红 (NOT panic)
//   - Error:       runtime fail (Run 返 error) → 节点变红
//   - Panic:       framework invariant broken (double Fire / Out(unknown) / impossible state) → recover + stack 进 log
func RunNode(ctx context.Context, rn *RegisteredNode, dataWire, config, execData map[string]any, vision VisionService, log LogService) RunResult {
	// Build inputs (priority: data-wire > config > exec-data > default)
	defaults := defaultsFromSpec(&rn.Spec)
	in := newInputs(dataWire, config, execData, defaults)

	// Phase 1: Required pre-Run check (ValidationError 非 panic — GPT r4 #8)
	if errs := validateRequired(&rn.Spec, in); len(errs) > 0 {
		return RunResult{Validation: errs}
	}

	// Phase 2: 节点自身 Validate (if Validator)
	if rn.Validate != nil {
		if errs := rn.Validate(in); len(errs) > 0 {
			return RunResult{Validation: errs}
		}
	}

	// Phase 3: 实际 Run, recover panic
	c := newCtx(ctx, vision, log, &rn.Spec)
	result := RunResult{}
	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Panic = r
				result.PanicStack = string(debug.Stack())
			}
		}()
		outs, err := rn.Impl.Run(c, in)
		if err != nil {
			result.Error = err
			return
		}
		if outs != nil {
			name, data := outs.exit()
			result.ExitName = name
			result.OutputData = data

			// Phase 4: Display (if Displayer)
			if rn.Display != nil {
				od := &outputDataImpl{data: data}
				result.DisplayText = rn.Display(in, name, od)
			}
		}
	}()
	return result
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
		if input.Type == "Exec" {
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
