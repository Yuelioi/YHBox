package runtime

import (
	"testing"

	"yotta/internal/services/container"
)

func TestExecState_NewState(t *testing.T) {
	s := NewExecState("container-1", 1800)
	if s.RunID == "" {
		t.Errorf("RunID should be auto-generated UUID")
	}
	if s.CalibCounts != 1800 {
		t.Errorf("CalibCounts not set: %d", s.CalibCounts)
	}
	if s.CurrentFrame == nil {
		t.Errorf("CurrentFrame should be initial main frame")
	}
	if !s.CurrentFrame.Graph.IsMain() {
		t.Errorf("initial frame should be main")
	}
	if s.CurrentFrame.ContainerID != "container-1" {
		t.Errorf("ContainerID lost: %q", s.CurrentFrame.ContainerID)
	}
}

func TestExecState_PushPopFrame(t *testing.T) {
	s := NewExecState("c1", 1800)
	sgRef := &container.Subgraph{ID: "sg-A", Label: "A"}
	s.PushFrame(container.SubgraphGraphRef("sg-A"), sgRef, "call-A")
	if s.CurrentFrame.Graph.ID != "sg-A" {
		t.Errorf("after push, current = %+v", s.CurrentFrame.Graph)
	}
	if s.CurrentFrame.Parent == nil || !s.CurrentFrame.Parent.Graph.IsMain() {
		t.Errorf("parent missing")
	}

	s.PopFrame()
	if !s.CurrentFrame.Graph.IsMain() {
		t.Errorf("after pop, should return to main")
	}
}

func TestExecFrame_LocalVarsIsolated(t *testing.T) {
	s := NewExecState("c1", 1800)
	s.SetLocalVar("x", 1)
	sgRef := &container.Subgraph{ID: "sg-A"}
	s.PushFrame(container.SubgraphGraphRef("sg-A"), sgRef, "call-A")
	if _, ok := s.GetLocalVarHere("x"); ok {
		t.Errorf("new frame should have empty LocalVars")
	}
	s.SetLocalVar("x", 100)
	if v, _ := s.GetLocalVarHere("x"); v != 100 {
		t.Errorf("child frame x = %v want 100", v)
	}
	s.PopFrame()
	if v, _ := s.GetLocalVarHere("x"); v != 1 {
		t.Errorf("main frame x = %v want 1", v)
	}
}

func TestExecState_ResolveVarFallback(t *testing.T) {
	s := NewExecState("c1", 1800)
	s.GlobalVars["gv"] = "global"
	sgRef := &container.Subgraph{ID: "sg"}
	s.PushFrame(container.SubgraphGraphRef("sg"), sgRef, "call-sg")
	s.SetLocalVar("lv", "local")

	if v, _ := s.ResolveVar("lv"); v != "local" {
		t.Errorf("ResolveVar(lv) = %v want local", v)
	}
	if v, _ := s.ResolveVar("gv"); v != "global" {
		t.Errorf("ResolveVar(gv) = %v want global", v)
	}
	if _, ok := s.ResolveVar("missing"); ok {
		t.Errorf("missing should not be found")
	}
}

func TestExecState_ResolveSourceCalibCounts(t *testing.T) {
	// 主图 (无 SubgraphRef) → 0（不缩放）
	s := NewExecState("c1", 2200)
	if got := s.ResolveSourceCalibCounts(); got != 0 {
		t.Errorf("main frame source = %d want 0", got)
	}

	// 手动组装的 subgraph (RecordingContext == nil) → 0
	manualSG := &container.Subgraph{ID: "sg-manual"}
	s.PushFrame(container.SubgraphGraphRef("sg-manual"), manualSG, "call-manual")
	if got := s.ResolveSourceCalibCounts(); got != 0 {
		t.Errorf("manual sg source = %d want 0", got)
	}
	s.PopFrame()

	// 录制 subgraph 带 RecordingContext → 返该值
	recSG := &container.Subgraph{
		ID:               "sg-rec",
		RecordingContext: &container.RecordingContext{MouseCounts360: 4000},
	}
	s.PushFrame(container.SubgraphGraphRef("sg-rec"), recSG, "call-rec")
	if got := s.ResolveSourceCalibCounts(); got != 4000 {
		t.Errorf("rec sg source = %d want 4000", got)
	}

	// 嵌套：内层 subgraph 无 RecordingContext，应沿 parent chain 拿到 4000
	innerSG := &container.Subgraph{ID: "sg-inner"}
	s.PushFrame(container.SubgraphGraphRef("sg-inner"), innerSG, "call-inner")
	if got := s.ResolveSourceCalibCounts(); got != 4000 {
		t.Errorf("nested inner source = %d want 4000 (from parent rec sg)", got)
	}
}

func TestExecState_PushNestedTwo(t *testing.T) {
	s := NewExecState("c1", 1800)
	sg1 := &container.Subgraph{ID: "sg1"}
	sg2 := &container.Subgraph{ID: "sg2"}
	s.PushFrame(container.SubgraphGraphRef("sg1"), sg1, "call-sg1")
	s.PushFrame(container.SubgraphGraphRef("sg2"), sg2, "call-sg2")
	if s.CurrentFrame.Graph.ID != "sg2" {
		t.Errorf("top frame should be sg2, got %s", s.CurrentFrame.Graph.ID)
	}
	s.PopFrame()
	if s.CurrentFrame.Graph.ID != "sg1" {
		t.Errorf("after one pop, should be sg1")
	}
	s.PopFrame()
	if !s.CurrentFrame.Graph.IsMain() {
		t.Errorf("after two pops, should be main")
	}
}
