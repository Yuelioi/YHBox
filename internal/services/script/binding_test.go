package script

import (
	"errors"
	"strings"
	"testing"

	"github.com/dop251/goja"

	"yotta/internal/node"
)

func TestNormalizeJS_Int64ToFloat64(t *testing.T) {
	got := NormalizeJS(map[string]any{"a": int64(3), "b": []any{int64(1), "s"}})
	m := got.(map[string]any)
	if m["a"] != float64(3) || m["b"].([]any)[0] != float64(1) {
		t.Fatalf("got %#v", got)
	}
}

func TestAdjustWrapLine(t *testing.T) {
	if got := AdjustWrapLine("SyntaxError: script:3:1 ..."); !strings.Contains(got, "script:2") {
		t.Fatalf("got %q", got)
	}
}

func TestCheckSyntax(t *testing.T) {
	if err := CheckSyntax("let a = 1; return a"); err != nil {
		t.Fatalf("valid script: %v", err) // IIFE 包裹后顶层 return 必须合法
	}
	if err := CheckSyntax("let a = ;"); err == nil {
		t.Fatal("want syntax error")
	}
}

// --- ScriptBindable 判定 ---

// fakeRunnable — 有 Run capability 的普通 exec 节点.
type fakeRunnable struct{ kind string }

func (f fakeRunnable) Spec() node.Spec { return node.Spec{Kind: f.kind, Category: "Control"} }
func (fakeRunnable) Run(node.Ctx, node.Inputs) (node.Outputs, error) {
	return nil, nil
}

// fakeEvaluator — 纯数据节点.
type fakeEvaluator struct{}

func (fakeEvaluator) Spec() node.Spec { return node.Spec{Kind: "FakeEval", Category: "PureFunc"} }
func (fakeEvaluator) Evaluate(node.Ctx, node.Inputs) (any, error) {
	return nil, nil
}

// fakeMarker — 图结构标记节点, 零 capability.
type fakeMarker struct{}

func (fakeMarker) Spec() node.Spec {
	return node.Spec{Kind: "FakeMarker", Category: "Control", IsGraphMarker: true}
}

// fakeVisual — 纯视觉节点, 零 capability.
type fakeVisual struct{}

func (fakeVisual) Spec() node.Spec {
	return node.Spec{Kind: "FakeVisual", Category: "Control", IsVisualOnly: true}
}

func TestScriptBindable(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(fakeRunnable{kind: "FakeRun"})
	node.Register(fakeEvaluator{})
	node.Register(fakeMarker{})
	node.Register(fakeVisual{})
	node.Register(fakeRunnable{kind: "Script"}) // Script 自身防递归

	want := map[string]bool{
		"FakeRun":    true,
		"FakeEval":   true,
		"FakeMarker": false,
		"FakeVisual": false,
		"Script":     false,
	}
	for _, rn := range node.All() {
		if got := node.ScriptBindable(rn); got != want[rn.Spec.Kind] {
			t.Fatalf("ScriptBindable(%s) = %v, want %v", rn.Spec.Kind, got, want[rn.Spec.Kind])
		}
	}
}

// --- goja 注意点 1 实测: panic(vm.ToValue(obj)) 等价 JS throw ---

func TestThrowErr_CatchableInJS(t *testing.T) {
	vm := goja.New()
	_ = vm.Set("boom", func(call goja.FunctionCall) goja.Value {
		throwErr(vm, "Fake", node.CodeTimeout, "fake timeout")
		return nil // unreachable
	})

	// 脚本 try/catch 能接到, e.code/e.kind 字段可读.
	v, err := vm.RunString(`(function(){ try { boom() } catch (e) { return e.code + ":" + e.kind } })()`)
	if err != nil {
		t.Fatalf("catch path: %v", err)
	}
	if got := v.Export(); got != "timeout:Fake" {
		t.Fatalf("got %#v", got)
	}

	// 未 catch → *goja.Exception, Value().Export() 是带 code 的 map (Task 4 mapScriptError 依赖此形态).
	_, err = vm.RunString(`boom()`)
	var jsErr *goja.Exception
	if !errors.As(err, &jsErr) {
		t.Fatalf("want *goja.Exception, got %T %v", err, err)
	}
	m, ok := jsErr.Value().Export().(map[string]any)
	if !ok || m["code"] != "timeout" || m["message"] != "fake timeout" {
		t.Fatalf("exception value: %#v", jsErr.Value().Export())
	}
}
