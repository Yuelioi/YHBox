package script

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dop251/goja"

	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/internal/services/llm"
)

type installTestCtx struct{ services node.ServiceBundle }

func (c installTestCtx) Context() context.Context            { return context.Background() }
func (installTestCtx) Now() time.Time                        { return time.Unix(0, 0) }
func (c installTestCtx) Services() node.ServiceBundle        { return c.services }
func (installTestCtx) Out(string) node.OutBuilder            { return nil }
func (installTestCtx) CaptureOutput(field string, value any) {}

type bundleSubgraphs struct{}

func (*bundleSubgraphs) CallSubgraph(context.Context, string, map[string]any) (string, error) {
	return "", nil
}

type bundleAI struct{}

func (*bundleAI) Provider(string) (llm.Provider, error) { return nil, nil }

func TestBundleFromCtx_ForwardsEveryNodeService(t *testing.T) {
	want := node.StubServices()
	want.Subgraphs = &bundleSubgraphs{}
	want.AI = &bundleAI{}
	want.Snapshot = func(context.Context) node.Snapshot { return node.Snapshot{} }
	got := BundleFromCtx(installTestCtx{services: want})
	fields := []struct {
		name      string
		got, want any
	}{
		{"Log", got.Log, want.Log},
		{"Input", got.Input, want.Input}, {"Vars", got.Vars, want.Vars},
		{"Params", got.Params, want.Params}, {"Window", got.Window, want.Window},
		{"Target", got.Target, want.Target}, {"App", got.App, want.App},
		{"Capture", got.Capture, want.Capture}, {"Stopwatches", got.Stopwatches, want.Stopwatches},
		{"Subgraphs", got.Subgraphs, want.Subgraphs},
		{"AI", got.AI, want.AI}, {"Registry", got.Registry, want.Registry},
	}
	for _, field := range fields {
		if !reflect.DeepEqual(field.got, field.want) {
			t.Errorf("%s was not forwarded: got=%T want=%T", field.name, field.got, field.want)
		}
	}
	if got.Snapshot != nil {
		t.Fatal("BundleFromCtx must not forward the framework-internal tick Snapshot hook")
	}
}

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

type fakePointRunnable struct{}

func (fakePointRunnable) Spec() node.Spec {
	return node.Spec{
		Kind:     "FakePointRun",
		Category: "Input",
		Inputs: []node.InputSpec{
			{Name: "Point", Type: "Point", Required: true},
		},
		Outputs: []node.OutputSpec{
			{Name: "Done", Type: "Exec", Data: []node.DataField{
				{Name: "X", Type: "Number"},
				{Name: "Y", Type: "Number"},
				{Name: "Unit", Type: "String"},
			}},
		},
	}
}

func (fakePointRunnable) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	pt := in.Point("Point")
	return ctx.Out("Done").
		Set("X", pt.X).
		Set("Y", pt.Y).
		Set("Unit", string(pt.Unit)).
		Fire(), nil
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
	registry := node.NewRegistry()
	registry.Register(fakeRunnable{kind: "FakeRun"})
	registry.Register(fakeEvaluator{})
	registry.Register(fakeMarker{})
	registry.Register(fakeVisual{})
	registry.Register(fakeRunnable{kind: "Script"}) // Script 自身防递归

	want := map[string]bool{
		"FakeRun":    true,
		"FakeEval":   true,
		"FakeMarker": false,
		"FakeVisual": false,
		"Script":     false,
	}
	for _, rn := range registry.All() {
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

func TestInstall_InjectsExitConstants(t *testing.T) {
	registry := node.NewRegistry()
	services := node.StubServices()
	services.Registry = registry.Snapshot()
	vm := goja.New()
	Install(vm, installTestCtx{services: services})

	v, err := vm.RunString(`Exit.Done + "|" + Exit.NotFound + "|" + Exit.Default`)
	if err != nil {
		t.Fatalf("read Exit constants: %v", err)
	}
	if got := v.String(); got != "Done|NotFound|default" {
		t.Fatalf("Exit constants = %q", got)
	}
}

func TestBindNode_CoercesObjectPinsBySpecType(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(fakePointRunnable{})
	services := node.StubServices()
	services.Registry = registry.Snapshot()

	vm := goja.New()
	Install(vm, installTestCtx{services: services})

	v, err := vm.RunString(`(function(){
		const r = FakePointRun({Point: {x: 12, y: 34, unit: "px"}})
		return r.exit + "|" + r.X + "|" + r.Y + "|" + r.Unit
	})()`)
	if err != nil {
		t.Fatalf("FakePointRun script call: %v", err)
	}
	if got := v.String(); got != "Done|12|34|px" {
		t.Fatalf("script point coercion = %q, want Done|12|34|px", got)
	}
}
