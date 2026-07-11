// internal/services/script/binding.go
// 节点即函数绑定层: 每个 ScriptBindable 节点 = vm 里同名全局函数.
// 调用约定: 单对象参数 {Pin: 值}; exec 节点返 {exit, ...出口Data}; PureFunc 直接返值.
// 节点失败 = 带 code 的 JS 异常 (脚本 try/catch 可接).
package script

import (
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/dop251/goja"

	"github.com/yottaapp/yotta/internal/node"
)

// BundleFromCtx 从宿主 Ctx 重组 ServiceBundle. Snapshot 留 nil —
// 脚本是 exec 语境, 读 live 变量是正确语义 (同 Runnable 节点).
func BundleFromCtx(c node.Ctx) node.ServiceBundle {
	bundle := c.Services()
	bundle.Snapshot = nil
	return bundle
}

// Install 注入标准出口常量 + 全部可绑节点函数 + Subgraph() 子图调用 + 高频糖 (params/sleep/log).
func Install(vm *goja.Runtime, c node.Ctx) {
	bundle := BundleFromCtx(c)
	installExitConstants(vm)
	for _, rn := range c.Services().Registry.All() {
		if node.ScriptBindable(rn) {
			bindNode(vm, c, rn, bundle)
		}
	}
	installSubgraphCall(vm, c)
	installSugar(vm, c)
	installVarGetters(vm, c)
}

func installExitConstants(vm *goja.Runtime) {
	exit := vm.NewObject()
	for key, value := range map[string]string{
		"Body":     "Body",
		"Changed":  "Changed",
		"Default":  "default",
		"Done":     "Done",
		"Fail":     "Fail",
		"False":    "False",
		"Found":    "Found",
		"Gone":     "Gone",
		"NotFound": "NotFound",
		"Out":      "Out",
		"Stable":   "Stable",
		"Timeout":  "Timeout",
		"True":     "True",
	} {
		_ = exit.Set(key, value)
	}
	_ = vm.Set("Exit", exit)
}

// installSubgraphCall 注入 Subgraph({SubgraphID, ...入参}) — 同步跑容器内子图,
// 返 {exit: <出口名>}. 走 runner 注入的 node.SubgraphCaller; 非 runner 语境 (单节点
// 测试) 服务为 nil, 不装 — 脚本里调用得到 ReferenceError, 与未知函数同语义.
// 失败 = 带 code 的 JS 异常 (同节点函数错误约定); 容器停止由 Script 节点 watchdog 兜底打断.
func installSubgraphCall(vm *goja.Runtime, c node.Ctx) {
	caller := c.Services().Subgraphs
	if caller == nil {
		return
	}
	vm.Set("Subgraph", func(call goja.FunctionCall) goja.Value {
		args := exportArg(vm, "Subgraph", call)
		sgID, _ := args["SubgraphID"].(string)
		if sgID == "" {
			throwErr(vm, "Subgraph", node.CodeError, `Subgraph(...) 需要字符串 SubgraphID`)
		}
		params := make(map[string]any, len(args))
		for k, v := range args {
			if k != "SubgraphID" {
				params[k] = v
			}
		}
		exit, err := caller.CallSubgraph(c.Context(), sgID, params)
		if err != nil {
			code := node.CodeError
			var coded node.Coded
			if errors.As(err, &coded) {
				code = coded.ErrCode()
			}
			throwErr(vm, "Subgraph", code, err.Error())
		}
		return vm.ToValue(map[string]any{"exit": exit})
	})
}

func bindNode(vm *goja.Runtime, c node.Ctx, rn *node.RegisteredNode, bundle node.ServiceBundle) {
	kind := rn.Spec.Kind
	if rn.Evaluate != nil {
		vm.Set(kind, func(call goja.FunctionCall) goja.Value {
			args := node.CoerceInputMap(&rn.Spec, exportArg(vm, kind, call))
			v, err := node.EvaluatePureData(c.Context(), rn, args, nil, bundle)
			if err != nil {
				throwErr(vm, kind, node.CodeError, err.Error())
			}
			return vm.ToValue(v)
		})
		return
	}
	vm.Set(kind, func(call goja.FunctionCall) goja.Value {
		args := node.CoerceInputMap(&rn.Spec, exportArg(vm, kind, call))
		res := node.RunNode(c.Context(), rn, args, nil, nil, bundle, false)
		if res.Panic != nil {
			panic(res.Panic) // 框架不变量破坏 — 不许被脚本 catch, 上抛给宿主 runWithRecover
		}
		if len(res.Validation) > 0 {
			msgs := make([]string, 0, len(res.Validation))
			for _, e := range res.Validation {
				msgs = append(msgs, e.Message)
			}
			throwErr(vm, kind, node.CodeError, strings.Join(msgs, "; "))
		}
		if res.Error != nil {
			code := node.CodeError
			var coded node.Coded
			if errors.As(res.Error, &coded) {
				code = coded.ErrCode()
			}
			throwErr(vm, kind, code, res.Error.Error())
		}
		out := map[string]any{"exit": res.ExitName}
		maps.Copy(out, res.OutputData)
		return vm.ToValue(out)
	})
}

// exportArg 首参对象 → map (缺省空 map = 全走 Default); 非对象 → throw.
// goja Export 整数是 int64, Inputs coercion 不认 — 递归归一成 float64.
func exportArg(vm *goja.Runtime, kind string, call goja.FunctionCall) map[string]any {
	if len(call.Arguments) == 0 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
		return map[string]any{}
	}
	m, ok := NormalizeJS(call.Arguments[0].Export()).(map[string]any)
	if !ok {
		throwErr(vm, kind, node.CodeError, fmt.Sprintf("%s(...) expects an object argument {Pin: value}", kind))
	}
	return m
}

// NormalizeJS 递归把 goja Export 的 int64 归一成 float64 (JS number 语义).
func NormalizeJS(v any) any {
	switch t := v.(type) {
	case int64:
		return float64(t)
	case []any:
		for i := range t {
			t[i] = NormalizeJS(t[i])
		}
		return t
	case map[string]any:
		for k := range t {
			t[k] = NormalizeJS(t[k])
		}
		return t
	default:
		return v
	}
}

// throwErr 抛带 code 的 JS 异常 (goja: panic 一个 goja value = JS throw).
func throwErr(vm *goja.Runtime, kind string, code node.ErrCode, msg string) {
	obj := vm.NewObject()
	_ = obj.Set("name", "NodeError")
	_ = obj.Set("kind", kind)
	_ = obj.Set("code", string(code))
	_ = obj.Set("message", msg)
	panic(vm.ToValue(obj))
}

// varNamer — VarStore 的可选能力: 枚举已知变量名 (生产 varStoreAdapter / 测试 stub 实现).
// 不进 node.VarStore 主接口 — 只有 $getter 注入需要, 测试 fake 不必陪绑.
type varNamer interface{ Names() []string }

// installVarGetters 给每个已知变量注入 $name live getter — 脚本里 $hp 实时读
// auto scope 变量。运行中动态新建的变量没有 getter (脚本起跑后注入集固定),
// 用 GetVar({VarName}) 节点函数读。
func installVarGetters(vm *goja.Runtime, c node.Ctx) {
	nv, ok := c.Services().Vars.(varNamer)
	if !ok {
		return
	}
	global := vm.GlobalObject()
	for _, name := range nv.Names() {
		n := name
		getter := vm.ToValue(func() goja.Value {
			v, _ := c.Services().Vars.GetScoped(n, "auto")
			return vm.ToValue(NormalizeJS(v))
		})
		_ = global.DefineAccessorProperty("$"+n, getter, nil, goja.FLAG_FALSE, goja.FLAG_TRUE)
	}
}

func installSugar(vm *goja.Runtime, c node.Ctx) {
	params := vm.NewObject()
	_ = params.Set("get", func(name string) goja.Value {
		v, _ := c.Services().Params.Get(name)
		return vm.ToValue(v)
	})
	_ = vm.Set("params", params)

	// 可取消 sleep: 容器停止时立即返回, watchdog 的 Interrupt 紧随其后掐断脚本.
	_ = vm.Set("sleep", func(ms float64) {
		if ms <= 0 {
			return
		}
		select {
		case <-c.Context().Done():
		case <-time.After(time.Duration(ms) * time.Millisecond):
		}
	})

	logObj := vm.NewObject()
	_ = logObj.Set("debug", func(args ...any) { c.Services().Log.Debug("%s", sprintArgs(args)) })
	_ = logObj.Set("info", func(args ...any) { c.Services().Log.Info("%s", sprintArgs(args)) })
	_ = logObj.Set("warn", func(args ...any) { c.Services().Log.Warn("%s", sprintArgs(args)) })
	_ = vm.Set("log", logObj)
}

func sprintArgs(args []any) string {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = fmt.Sprint(NormalizeJS(a))
	}
	return strings.Join(parts, " ")
}
