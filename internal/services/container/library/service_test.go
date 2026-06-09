package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"yotta/internal/services/asset"
	"yotta/internal/services/container"
)

// TestService_ImportToContainer_SyncsContainerStore 复现真 bug:
// import 直接写 subgraphs/*.json 绕过容器 Store 内存缓存, 不 reload 则 ListSubgraphs
// 返旧值, 刚导入的子图查不到 → 前端节点 "(子图未找到)".
// SetContainerReloader 注入后 import 应让 containerStore 立刻看见新子图.
func TestService_ImportToContainer_SyncsContainerStore(t *testing.T) {
	// 1) library bundle "foo" (root subgraph id=foo + 一条 template 资产).
	libRoot := t.TempDir()
	as := newAssetStore(t)
	sha, _ := as.Blobs().Put([]byte("pixels"))
	rec := asset.AssetRecord{
		GUID: "g1", Kind: asset.KindTemplate, Name: "登录",
		Variants: []asset.Variant{{Resolution: [2]int{1920, 1080}, Blob: sha}},
	}
	writeBundlePkg(t, libRoot, "foo", rec, map[string][]byte{sha: []byte("pixels")})

	// 2) 目标容器 ctrB 落盘 + 起容器 Store (内存初始无子图).
	containersRoot := t.TempDir()
	const ctrID = "ctrB"
	os.MkdirAll(filepath.Join(containersRoot, ctrID), 0o755)
	cj := container.Container{
		SchemaVersion: container.CurrentSchemaVersion, ID: ctrID, Name: "ctrB",
		Graph: container.Graph{ID: "g", Version: container.GraphSchemaVersion},
	}
	cb, _ := json.Marshal(cj)
	os.WriteFile(filepath.Join(containersRoot, ctrID, "container.json"), cb, 0o644)

	cs, err := container.NewStore(containersRoot)
	if err != nil {
		t.Fatal(err)
	}
	if sgs := cs.ListSubgraphs(ctrID); len(sgs) != 0 {
		t.Fatalf("precondition: container should start with 0 subgraphs, got %d", len(sgs))
	}

	// 3) library Service + reloader 接 containerStore.Reload.
	libStore, _ := NewStore(libRoot)
	svc := NewService(libStore, as)
	svc.SetContainersRoot(containersRoot)
	svc.SetContainerReloader(func(id string) error { _, err := cs.Reload(id); return err })

	if _, err := svc.ImportToContainer("foo", ctrID); err != nil {
		t.Fatal(err)
	}

	// 4) 关键断言: import 后容器 Store 内存里能查到刚导入的子图.
	//    (修复前: reload 缺失, in-memory 仍空, GetSubgraph 返 false → 节点 "子图未找到".)
	if _, ok := cs.GetSubgraph(ctrID, "foo"); !ok {
		t.Error("imported subgraph 'foo' not visible via containerStore — ListSubgraphs would miss it (FE: 子图未找到)")
	}
}
