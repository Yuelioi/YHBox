package container

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestSubgraphStore(t *testing.T) (*SubgraphStore, string) {
	t.Helper()
	root := t.TempDir()
	st, err := NewSubgraphStore(root)
	if err != nil {
		t.Fatal(err)
	}
	return st, root
}

func TestSubgraphStore_CreateAndGet_RoundTrip(t *testing.T) {
	st, root := newTestSubgraphStore(t)
	sg := &Subgraph{
		ID:    "sg-A",
		Label: "A",
		Graph: Graph{
			ID: "g-sgA", SchemaVersion: GraphSchemaVersion,
			Nodes: []GraphNode{
				// 老格式 marker 节点 — persist 路径 normalize 应提取到 metadata 并 strip.
				{ID: "in", Kind: "SubgraphInput", CreatedAt: time.Now().UTC()},
				{ID: "out", Kind: "SubgraphOutput", CreatedAt: time.Now().UTC()},
				// global var 引用 — persist 应派生进 RequiredGlobals.
				{ID: "gv", Kind: "GetVar", Config: map[string]any{"VarName": "state", "Scope": "auto"}, CreatedAt: time.Now().UTC()},
			},
		},
		OutputPins: []SubgraphOutputDecl{{ID: "d", Name: "done"}},
		CreatedAt:  time.Now().UTC(),
	}
	if err := st.Create(sg); err != nil {
		t.Fatalf("create: %v", err)
	}
	if sg.Rev != 1 {
		t.Errorf("Rev = %d, want 1", sg.Rev)
	}
	got, ok := st.Get("sg-A")
	if !ok {
		t.Fatal("not found after Create")
	}
	if got.Label != "A" {
		t.Errorf("label = %q", got.Label)
	}
	// normalize self-heal: marker 节点 strip 进 metadata.
	if got.Entry.NodeID != "in" {
		t.Errorf("Entry.NodeID = %q, want \"in\" (migrated from SubgraphInput node)", got.Entry.NodeID)
	}
	if len(got.OutputPins) != 1 || got.OutputPins[0].NodeID != "out" {
		t.Errorf("OutputPins not reconciled from SubgraphOutput node: %+v", got.OutputPins)
	}
	for _, n := range got.Graph.Nodes {
		if n.Kind == "SubgraphInput" || n.Kind == "SubgraphOutput" {
			t.Errorf("marker node %q should be stripped after normalize", n.ID)
		}
	}
	// RequiredGlobals 保存时派生 (只存名字).
	if len(got.RequiredGlobals) != 1 || got.RequiredGlobals[0] != "state" {
		t.Errorf("RequiredGlobals = %+v, want [state]", got.RequiredGlobals)
	}
	if got.SchemaVersion != SubgraphSchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", got.SchemaVersion, SubgraphSchemaVersion)
	}
	// 文件平铺在 <root>/<id>.json.
	if _, err := os.Stat(filepath.Join(root, "sg-A.json")); err != nil {
		t.Errorf("expected flat file <root>/sg-A.json: %v", err)
	}

	// 重新打开 store — 磁盘 round-trip.
	st2, err := NewSubgraphStore(root)
	if err != nil {
		t.Fatal(err)
	}
	got2, ok := st2.Get("sg-A")
	if !ok {
		t.Fatal("not found after reload")
	}
	if got2.Label != "A" || got2.Rev != 1 {
		t.Errorf("reload mismatch: label=%q rev=%d", got2.Label, got2.Rev)
	}
}

func TestSubgraphStore_Create_MintsIDWhenEmpty(t *testing.T) {
	st, _ := newTestSubgraphStore(t)
	sg := &Subgraph{Label: "anon"}
	if err := st.Create(sg); err != nil {
		t.Fatalf("create: %v", err)
	}
	if !strings.HasPrefix(sg.ID, "sg-") {
		t.Errorf("minted ID %q should have sg- prefix", sg.ID)
	}
	if sg.Rev != 1 {
		t.Errorf("Rev = %d, want 1", sg.Rev)
	}
	if _, ok := st.Get(sg.ID); !ok {
		t.Error("minted subgraph not retrievable")
	}
}

func TestSubgraphStore_Create_RejectsExistingID(t *testing.T) {
	st, _ := newTestSubgraphStore(t)
	if err := st.Create(&Subgraph{ID: "sg-dup", Label: "first"}); err != nil {
		t.Fatal(err)
	}
	if err := st.Create(&Subgraph{ID: "sg-dup", Label: "second"}); err == nil {
		t.Error("Create with existing ID should error")
	}
}

func TestSubgraphStore_List(t *testing.T) {
	st, _ := newTestSubgraphStore(t)
	for _, id := range []string{"sg-1", "sg-2", "sg-3"} {
		if err := st.Create(&Subgraph{ID: id, Label: id}); err != nil {
			t.Fatal(err)
		}
	}
	list := st.List()
	if len(list) != 3 {
		t.Errorf("expected 3, got %d", len(list))
	}
}

func TestSubgraphStore_Update_HappyPath(t *testing.T) {
	st, _ := newTestSubgraphStore(t)
	sg := &Subgraph{ID: "sg-u", Label: "v1"}
	if err := st.Create(sg); err != nil {
		t.Fatal(err)
	}
	upd := *sg
	upd.Label = "v2"
	if err := st.Update(&upd, 1); err != nil {
		t.Fatalf("update: %v", err)
	}
	if upd.Rev != 2 {
		t.Errorf("Rev after update = %d, want 2", upd.Rev)
	}
	got, _ := st.Get("sg-u")
	if got.Label != "v2" || got.Rev != 2 {
		t.Errorf("got label=%q rev=%d, want v2/2", got.Label, got.Rev)
	}
}

func TestSubgraphStore_Update_StaleRevRejected(t *testing.T) {
	st, _ := newTestSubgraphStore(t)
	sg := &Subgraph{ID: "sg-s", Label: "v1"}
	if err := st.Create(sg); err != nil {
		t.Fatal(err)
	}
	upd := *sg
	upd.Label = "v2"
	if err := st.Update(&upd, 1); err != nil {
		t.Fatal(err)
	}
	// 第二个编辑器还拿着 baseRev=1 → 拒绝.
	stale := *sg
	stale.Label = "conflict"
	err := st.Update(&stale, 1)
	if !errors.Is(err, ErrSubgraphRevStale) {
		t.Fatalf("expected ErrSubgraphRevStale, got %v", err)
	}
	got, _ := st.Get("sg-s")
	if got.Label != "v2" {
		t.Errorf("stale update should not land, got label=%q", got.Label)
	}
}

func TestSubgraphStore_Update_MissingErrors(t *testing.T) {
	st, _ := newTestSubgraphStore(t)
	err := st.Update(&Subgraph{ID: "sg-nope", Label: "x"}, 1)
	if err == nil {
		t.Error("Update on missing subgraph should error")
	}
}

func TestSubgraphStore_Delete_IdempotentAndStaleRev(t *testing.T) {
	st, root := newTestSubgraphStore(t)
	sg := &Subgraph{ID: "sg-d", Label: "x"}
	if err := st.Create(sg); err != nil {
		t.Fatal(err)
	}
	// stale rev → 拒绝.
	if err := st.Delete("sg-d", 99); !errors.Is(err, ErrSubgraphRevStale) {
		t.Fatalf("expected ErrSubgraphRevStale, got %v", err)
	}
	if _, ok := st.Get("sg-d"); !ok {
		t.Fatal("stale delete should not remove")
	}
	// 正确 rev → 删除.
	if err := st.Delete("sg-d", 1); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok := st.Get("sg-d"); ok {
		t.Error("still found after delete")
	}
	if _, err := os.Stat(filepath.Join(root, "sg-d.json")); !os.IsNotExist(err) {
		t.Error("file should be gone after delete")
	}
	// 不存在 → 视为成功 (idempotent).
	if err := st.Delete("sg-d", 1); err != nil {
		t.Errorf("delete of missing should succeed, got %v", err)
	}
	if err := st.Delete("sg-never-existed", 0); err != nil {
		t.Errorf("delete of never-existed should succeed, got %v", err)
	}
}

func TestSubgraphStore_Duplicate(t *testing.T) {
	st, _ := newTestSubgraphStore(t)
	orig := &Subgraph{ID: "sg-orig", Label: "录制", IsAnonymous: true}
	if err := st.Create(orig); err != nil {
		t.Fatal(err)
	}
	// 抬高原本 rev, 确认副本 rev 重置.
	upd := *orig
	if err := st.Update(&upd, 1); err != nil {
		t.Fatal(err)
	}

	dup, err := st.Duplicate("sg-orig")
	if err != nil {
		t.Fatalf("duplicate: %v", err)
	}
	if dup.ID == "sg-orig" || !strings.HasPrefix(dup.ID, "sg-") {
		t.Errorf("dup ID %q should be a freshly minted sg-<uuid>", dup.ID)
	}
	if dup.Rev != 1 {
		t.Errorf("dup Rev = %d, want 1", dup.Rev)
	}
	if dup.IsAnonymous {
		t.Error("dup should be named (IsAnonymous=false) — 用户主动 fork")
	}
	if dup.Label != "录制 (副本)" {
		t.Errorf("dup label = %q, want 录制 (副本)", dup.Label)
	}
	if _, ok := st.Get(dup.ID); !ok {
		t.Error("dup not retrievable from store")
	}
	// 原本不动.
	got, _ := st.Get("sg-orig")
	if got.Rev != 2 || !got.IsAnonymous {
		t.Errorf("original tampered by Duplicate: %+v", got)
	}
}

func TestSubgraphStore_Duplicate_MissingErrors(t *testing.T) {
	st, _ := newTestSubgraphStore(t)
	if _, err := st.Duplicate("sg-ghost"); err == nil {
		t.Error("Duplicate of missing subgraph should error")
	}
}

func TestSubgraphStore_GCAnonymous(t *testing.T) {
	st, root := newTestSubgraphStore(t)
	mk := func(id string, anon bool) {
		t.Helper()
		if err := st.Create(&Subgraph{ID: id, Label: id, IsAnonymous: anon}); err != nil {
			t.Fatal(err)
		}
	}
	mk("sg-anon-ref", true)    // 匿名 + 被引用 → 留
	mk("sg-anon-orphan", true) // 匿名 + 无引用 → 删
	mk("sg-named", false)      // 具名 → 永不自动删

	removed, err := st.GCAnonymous(map[string]bool{"sg-anon-ref": true})
	if err != nil {
		t.Fatalf("gc: %v", err)
	}
	if len(removed) != 1 || removed[0] != "sg-anon-orphan" {
		t.Errorf("removed = %v, want [sg-anon-orphan]", removed)
	}
	if _, ok := st.Get("sg-anon-orphan"); ok {
		t.Error("orphan anonymous should be gone")
	}
	if _, ok := st.Get("sg-anon-ref"); !ok {
		t.Error("referenced anonymous should survive")
	}
	if _, ok := st.Get("sg-named"); !ok {
		t.Error("named subgraph should survive")
	}
	if _, err := os.Stat(filepath.Join(root, "sg-anon-orphan.json")); !os.IsNotExist(err) {
		t.Error("orphan file should be removed from disk")
	}
}

func TestSubgraphStore_GenerationBumpsOnWrites(t *testing.T) {
	st, _ := newTestSubgraphStore(t)
	g0 := st.Generation()
	sg := &Subgraph{ID: "sg-gen", Label: "g"}
	if err := st.Create(sg); err != nil {
		t.Fatal(err)
	}
	g1 := st.Generation()
	if g1 <= g0 {
		t.Errorf("Generation should bump on Create: %d → %d", g0, g1)
	}
	if err := st.Delete("sg-gen", 1); err != nil {
		t.Fatal(err)
	}
	if g2 := st.Generation(); g2 <= g1 {
		t.Errorf("Generation should bump on Delete: %d → %d", g1, g2)
	}
}
