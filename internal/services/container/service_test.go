package container

import (
	"testing"
)

func TestContainerService_CreateAndList(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	svc := NewService(s)

	c, err := svc.Create("我的容器")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.ID == "" || c.Name != "我的容器" {
		t.Errorf("created: %+v", c)
	}
	list := svc.List()
	if len(list) != 1 {
		t.Errorf("expected 1, got %d", len(list))
	}
}

func TestContainerService_UpdateViaJSON(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	svc := NewService(s)
	c, _ := svc.Create("x")

	patchJSON := `{"name":"y","tags":["t1"]}`
	if err := svc.Update(c.ID, patchJSON); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := s.Get(c.ID)
	if got.Name != "y" || len(got.Tags) != 1 {
		t.Errorf("patch lost: %+v", got)
	}
}

func TestContainerService_Delete(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	svc := NewService(s)
	c, _ := svc.Create("x")
	if err := svc.Delete(c.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(svc.List()) != 0 {
		t.Error("not deleted")
	}
}

func TestContainerService_Update_PathTraversalProtected(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	svc := NewService(s)
	c, _ := svc.Create("x")
	originalID := c.ID

	// patch 试图改 ID
	patch := `{"id":"../../etc/evil","name":"hacked"}`
	if err := svc.Update(originalID, patch); err != nil {
		t.Fatalf("Update should succeed (ID override ignored): %v", err)
	}

	// 验证：原 ID 还在，name 已改，没有 ../../etc/evil 路径
	got, ok := s.Get(originalID)
	if !ok {
		t.Fatal("original container missing")
	}
	if got.Name != "hacked" {
		t.Errorf("name patch should still apply, got %q", got.Name)
	}
	if _, ok := s.Get("../../etc/evil"); ok {
		t.Error("malicious ID should not exist in store")
	}
}

func TestContainerStore_Save_InvalidID(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	bad := []string{"", "../foo", "a/b", "a\\b", ".", "..", "foo:bar"}
	for _, id := range bad {
		c := &Container{
			SchemaVersion: 1, ID: id, Name: "x",
			Graph: Graph{Nodes: []GraphNode{{ID: "n1", Kind: "Start"}}},
		}
		if err := s.Save(c); err == nil {
			t.Errorf("id %q should be rejected", id)
		}
	}
}

func TestCreate_SeedsBackendFieldsAndWiresWindowTarget(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	svc := NewService(s)

	c, err := svc.Create("t1")
	if err != nil {
		t.Fatal(err)
	}
	if c.InputBackend != "postmessage" || c.CaptureBackend != "auto" || c.ScaleTolerance < 1.0 {
		t.Fatalf("默认字段未 seed: InputBackend=%q CaptureBackend=%q ScaleTolerance=%v",
			c.InputBackend, c.CaptureBackend, c.ScaleTolerance)
	}

	var wtID string
	for _, n := range c.Graph.Nodes {
		if n.Kind == "WindowTarget" {
			wtID = n.ID
		}
	}
	if wtID == "" {
		t.Fatal("未找到 WindowTarget 节点")
	}

	hasStartToWT, hasWTToStop := false, false
	for _, e := range c.Graph.Edges {
		if e.To == wtID+".In" {
			hasStartToWT = true
		}
		if e.From == wtID+".Fire" {
			hasWTToStop = true
		}
	}
	if !hasStartToWT || !hasWTToStop {
		t.Fatalf("WindowTarget 未接进 Start→WindowTarget→Stop: edges=%+v", c.Graph.Edges)
	}
}
