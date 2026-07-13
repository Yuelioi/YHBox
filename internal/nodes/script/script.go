// internal/nodes/script/script.go
// Script — 内嵌 JS 脚本节点 (goja). 节点即函数: 全部 ScriptBindable 节点是脚本全局函数.
package script

import (
	"errors"
	"fmt"

	"github.com/dop251/goja"

	"github.com/yottaapp/yotta/internal/node"
	scriptsvc "github.com/yottaapp/yotta/internal/services/script"
)

func init() { node.Register(&Script{}) }

const (
	pinIn         = "In"
	pinDone       = "Done"
	pinFail       = "Fail"
	inCode        = "Code"
	outDataResult = "Result"
)

type Script struct{}

func (Script) Spec() node.Spec {
	return node.Spec{
		Kind:     "Script",
		Category: "Control",
		Inputs: []node.InputSpec{
			{Name: pinIn, Type: node.TypeExec},
			{Name: inCode, Type: "String", Default: "", Required: true,
				Widget: node.WidgetSpec{Kind: "code", Props: node.MarshalProps(node.TextareaProps{Rows: 8})}},
			node.WindowInputSpec(),
		},
		Outputs: []node.OutputSpec{
			{Name: pinDone, Type: node.TypeExec,
				Data: []node.DataField{{Name: outDataResult, Type: "*", Optional: true}}},
			{Name: pinFail, Type: node.TypeExec, Semantic: "error",
				Data: []node.DataField{{Name: "Error", Type: "String"}, {Name: "Code", Type: "String"}}},
		},
		NeedsWindow:     true, // 脚本可能调输入/视觉节点 — 保守要求 Win32WindowTarget (用户拍板 2026-06-10)
		NeedsForeground: true, // 脚本内调绑定节点可触发 SendInput — 需在派发时将 Window 提到前台
		RuntimeCapabilities: []node.RuntimeCapability{
			node.RuntimeCapabilityLog,
			node.RuntimeCapabilityParams,
			node.RuntimeCapabilityRegistry,
		},
		DynamicPorts: []node.DynamicPortSpec{{
			Role: node.DynamicPortInput, ConfigKey: "Inputs", Shape: node.DynamicPortNameTypeRecords,
			MaxItems: 256,
		}},
	}
}

// scriptEnvSkipKeys — merged inputs 里非动态输入的 key (静态 pin + config 元数据).
// "Window" 是静态 dispatch-override pin, 不应注入为 JS 全局变量。
var scriptEnvSkipKeys = map[string]struct{}{
	inCode: {}, "Inputs": {}, "Window": {},
}

// Dependencies 静态抽脚本里引用的资产 GUID — 让依赖扫描器 / 资产 GC / 安全删除看见脚本引用,
// 堵住"脚本引用的模板/clip 被 GC 误删、库里删不警告"的盲区。提取逻辑见 scriptsvc.AssetDeps。
func (Script) Dependencies(in node.Inputs) []node.Dependency {
	return scriptsvc.AssetDeps(in.String(inCode))
}

func (Script) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	src := in.String(inCode)
	prog, err := scriptsvc.CompileCached(src)
	if err != nil {
		// 配置错 (语法错): 裸 error 冒泡中断 (同 Expr parse 模式); 编辑期 SCRIPT_PARSE_ERROR 先拦.
		return nil, fmt.Errorf("Script parse: %s", scriptsvc.AdjustWrapLine(err.Error()))
	}

	vm := goja.New()
	scriptsvc.Install(vm, ctx)
	for _, k := range in.Keys() {
		if _, skip := scriptEnvSkipKeys[k]; skip {
			continue
		}
		_ = vm.Set(k, in.Raw(k))
	}

	// watchdog: 容器停止 → Interrupt 立即掐断脚本 (含纯 JS 死循环).
	watchDone := make(chan struct{})
	defer close(watchDone)
	go func() {
		select {
		case <-ctx.Context().Done():
			vm.Interrupt(ctx.Context().Err())
		case <-watchDone:
		}
	}()

	v, err := vm.RunProgram(prog)
	if err != nil {
		return nil, mapScriptError(ctx, err)
	}

	out := ctx.Out(pinDone)
	if v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
		result := scriptsvc.NormalizeJS(v.Export())
		out = out.Set(outDataResult, result)
	}
	return out.Fire(), nil
}

// mapScriptError — RunProgram 错误分类:
//
//	Interrupt (容器停止) → ctx.Err() graceful halt (同 Sleep/KeyPress);
//	JS 异常带 code (绑定层节点失败) → 透传原错误码走 Fail;
//	裸 throw → CodeThrown.
func mapScriptError(ctx node.Ctx, err error) error {
	var interrupted *goja.InterruptedError
	if errors.As(err, &interrupted) {
		if cerr := ctx.Context().Err(); cerr != nil {
			return cerr
		}
	}
	var jsErr *goja.Exception
	if errors.As(err, &jsErr) {
		if m, ok := jsErr.Value().Export().(map[string]any); ok {
			if code, _ := m["code"].(string); code != "" {
				msg, _ := m["message"].(string)
				return node.Failf(node.ErrCode(code), nil, "Script: %s", msg)
			}
		}
		return node.Failf(node.CodeThrown, nil, "Script throw: %s", scriptsvc.AdjustWrapLine(jsErr.Error()))
	}
	return node.Failf(node.CodeThrown, nil, "Script: %v", err)
}
