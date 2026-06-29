package variable

import (
	"context"
	"testing"

	"yotta/internal/node"
)

// recordingVars 实现 node.VarStore — 累积调用供测试断言.
type recordingVars struct {
	store map[string]any
}

func (r *recordingVars) Get(name string) (any, bool) {
	if r.store == nil {
		return nil, false
	}
	v, ok := r.store[name]
	return v, ok
}
func (r *recordingVars) Set(name string, value any) {
	if r.store == nil {
		r.store = map[string]any{}
	}
	r.store[name] = value
}
func (r *recordingVars) Inc(name string, delta float64) float64 {
	if r.store == nil {
		r.store = map[string]any{}
	}
	cur, _ := r.store[name].(float64)
	cur += delta
	r.store[name] = cur
	return cur
}

// scope-aware 变种: stub 不区分 frame, 全部走 global store. SetVar/IncVar 节点 5.5
// cutover 后调 *Scoped, recordingVars 透到 base Get/Set/Inc.
func (r *recordingVars) GetScoped(name, _ string) (any, bool)      { return r.Get(name) }
func (r *recordingVars) SetScoped(name, _ string, v any)            { r.Set(name, v) }
func (r *recordingVars) IncScoped(name, _ string, d float64) float64 { return r.Inc(name, d) }
func (r *recordingVars) LastChange(name string) int64                { return 0 }

func TestSetVar_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&SetVar{})
	rn, _ := node.Get("SetVar")
	vars := &recordingVars{}
	svc := node.StubServices()
	svc.Vars = vars
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{svInVarName: "x", svInValue: 42},
		nil, svc, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if vars.store["x"] != 42 {
		t.Errorf("vars[x] = %v, want 42", vars.store["x"])
	}
}

func TestSetVar_MissingVarName(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&SetVar{})
	rn, _ := node.Get("SetVar")
	svc := node.StubServices()
	svc.Vars = &recordingVars{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{svInVarName: "", svInValue: 1},
		nil, svc, false)
	// Required check 会先报 — varName 是 required + 空 → ValidationError
	if len(r.Validation) == 0 && r.Error == nil {
		t.Error("expected validation or error for empty varName")
	}
}
