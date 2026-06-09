package library

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"yotta/internal/services/asset"
	"yotta/internal/services/container"
	"yotta/internal/services/container/dependency"
)

// ImportResult Import 操作结果.
//
// Imported 列本次写入的依赖 (subgraph / asset). MissingGlobals 反映 import 的 sg 集合
// (Root+Embedded) RequiredGlobals union 减去目标 container.Vars 已声明的, 是 caller 需要补
// declare 的 var (跟资产 GUID 正交). FE 据此弹 "添加变量并继续" 提示.
type ImportResult struct {
	Imported       []dependency.Dependency            `json:"imported"`
	MissingGlobals []container.SubgraphRequiredGlobal `json:"missingGlobals,omitempty"`
}

// ImportToContainer 把 library package 幂等合并到容器 + 全局 asset 库.
//
// 资产: GUID 不存在 → 整条写入; 已存在 → 按 resolution 加性合并变体 (本地缺的 res 加入,
// 本地已有 res 保留本地; name/tags 保留本地). blob 按 sha 落池去重. clip (单 blob) 已存在则跳过.
// 图: subgraph 直接写入 (提交点). 无 conflict/strategy — GUID/sha 寻址天然幂等.
//
// containerVars 是目标容器当前已声明的 vars, 用来算 MissingGlobals diff (nil → 返 union 全集).
func (s *Store) ImportToContainer(libSgID, containerRoot string, assetStore *asset.Store, containerVars []container.VarDecl) (*ImportResult, error) {
	pkg, err := s.GetSubgraphPackage(libSgID)
	if err != nil {
		return nil, err
	}

	libPkgDir := filepath.Join(s.root, "subgraphs", libSgID)
	result := &ImportResult{}

	// 1) 资产合并 (record 加性 / blob 去重). 先于图写入 — 图是提交点.
	for _, guid := range pkg.Assets {
		recPath := filepath.Join(libPkgDir, "assets", guid+".json")
		b, err := os.ReadFile(recPath)
		if err != nil {
			return nil, fmt.Errorf("read bundle asset %q: %w", guid, err)
		}
		var rec asset.AssetRecord
		if err := json.Unmarshal(b, &rec); err != nil {
			return nil, fmt.Errorf("parse bundle asset %q: %w", guid, err)
		}
		if err := importAsset(assetStore, libPkgDir, rec); err != nil {
			return nil, err
		}
		depKind := dependency.KindTemplate
		if rec.Kind == asset.KindClip {
			depKind = dependency.KindClip
		}
		result.Imported = append(result.Imported, dependency.Dependency{Kind: depKind, Key: guid})
	}

	// 2) 图写入 (提交点). subgraph 直接覆盖写 (GUID 寻址, 幂等).
	containerSgDir := filepath.Join(containerRoot, "subgraphs")
	if err := os.MkdirAll(containerSgDir, 0o755); err != nil {
		return nil, err
	}
	writeSg := func(sg container.Subgraph) error {
		jb, _ := json.MarshalIndent(sg, "", "  ")
		if err := os.WriteFile(filepath.Join(containerSgDir, sg.ID+".json"), jb, 0o644); err != nil {
			return err
		}
		result.Imported = append(result.Imported, dependency.Dependency{Kind: dependency.KindSubgraph, Key: sg.ID})
		return nil
	}
	if err := writeSg(pkg.Root); err != nil {
		return nil, err
	}
	for _, sg := range pkg.Embedded {
		if err := writeSg(sg); err != nil {
			return nil, err
		}
	}

	result.MissingGlobals = computeMissingGlobals(pkg, containerVars)
	return result, nil
}

// importAsset 把 bundle 里一条资产记录幂等合并进全局 asset 库.
// blob 先按 sha 落池去重; record 按 GUID 加性合并 (见 ImportToContainer 注释).
func importAsset(store *asset.Store, libPkgDir string, rec asset.AssetRecord) error {
	// 收集本条记录引用的全部 blob sha → 从 bundle blobs/ 落池.
	putBlob := func(sha string) error {
		if sha == "" || store.Blobs().Has(sha) {
			return nil
		}
		data, err := os.ReadFile(filepath.Join(libPkgDir, "blobs", sha))
		if err != nil {
			return fmt.Errorf("read bundle blob %s: %w", sha, err)
		}
		got, err := store.Blobs().Put(data)
		if err != nil {
			return fmt.Errorf("put blob %s: %w", sha, err)
		}
		if got != sha {
			return fmt.Errorf("bundle blob sha mismatch: declared %s, computed %s", sha, got)
		}
		return nil
	}
	for _, v := range rec.Variants {
		if err := putBlob(v.Blob); err != nil {
			return err
		}
	}
	if err := putBlob(rec.Blob); err != nil {
		return err
	}

	existing, ok := store.Get(rec.GUID)
	if !ok {
		// 新资产 → 整条写入.
		return store.PutRecord(rec)
	}
	// clip (单 blob, 无变体): 已存在则保留本地, 不动.
	if rec.Kind == asset.KindClip {
		return nil
	}
	// template: 按 resolution 加性合并 — 本地缺的 res 加入, 本地已有 res 保留本地.
	have := map[[2]int]bool{}
	for _, v := range existing.Variants {
		have[v.Resolution] = true
	}
	for _, v := range rec.Variants {
		if have[v.Resolution] {
			continue // 本地已有该分辨率 → 保留本地, 忽略 bundle 同档.
		}
		if err := store.PutVariant(rec.GUID, v.Resolution, v.Blob, v.BBox, v.Regions); err != nil {
			return err
		}
	}
	return nil
}

// computeMissingGlobals 算 (pkg.Root + pkg.Embedded[*]).RequiredGlobals union 减去 containerVars
// 已声明的 var. 用于 import 前提示 caller 容器缺哪些 var.
// containerVars 为 nil → 等同于 caller 没声明任何 var, 返回 union 全集.
func computeMissingGlobals(pkg *SubgraphPackage, containerVars []container.VarDecl) []container.SubgraphRequiredGlobal {
	declared := map[string]bool{}
	for _, v := range containerVars {
		declared[v.Name] = true
	}
	seen := map[string]bool{}
	var out []container.SubgraphRequiredGlobal
	collect := func(sg container.Subgraph) {
		for _, rg := range sg.RequiredGlobals {
			if declared[rg.Name] || seen[rg.Name] {
				continue
			}
			seen[rg.Name] = true
			out = append(out, rg)
		}
	}
	collect(pkg.Root)
	for _, sg := range pkg.Embedded {
		collect(sg)
	}
	return out
}

// readSubgraph 从容器 subgraphs/ 读一个子图. 不存在返 (nil, nil).
func readSubgraph(srcContainerRoot, sgID string) (*container.Subgraph, error) {
	path := filepath.Join(srcContainerRoot, "subgraphs", sgID+".json")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	var sg container.Subgraph
	if err := json.Unmarshal(b, &sg); err != nil {
		return nil, err
	}
	return &sg, nil
}

// getNodesFromContainer 适配 dependency scanner 的 getNodes 回调.
func getNodesFromContainer(srcContainerRoot string) func(string) ([]dependency.NodeInfo, error) {
	return func(sgID string) ([]dependency.NodeInfo, error) {
		sg, err := readSubgraph(srcContainerRoot, sgID)
		if err != nil || sg == nil {
			return nil, err
		}
		nodes := make([]dependency.NodeInfo, len(sg.Graph.Nodes))
		for i, n := range sg.Graph.Nodes {
			nodes[i] = dependency.NodeInfo{Kind: n.Kind, Config: n.Config}
		}
		return nodes, nil
	}
}

// writeBundle 把 (rootSg + deps 闭包) 打成 library package: graphs + assets/<guid>.json + blobs/<sha256>.
// rootSg 是 bundle 根子图; pkgID 是 package 目录名. embedded = deps 里非 rootSg.ID 的 subgraph.
// 资产记录从全局 asset.Store 读; record 引用的 blob 缺失 / graph 引用的 GUID 无 record → 报错中止 (不写半包).
func (s *Store) writeBundle(srcContainerRoot, pkgID string, rootSg *container.Subgraph, deps []dependency.Dependency, assetStore *asset.Store, overwrite bool) error {
	pkgDir := filepath.Join(s.root, "subgraphs", pkgID)
	if !overwrite {
		if _, err := os.Stat(pkgDir); err == nil {
			return fmt.Errorf("library package %q already exists, use overwrite=true to replace", pkgID)
		}
	}

	// 先全量校验资产闭包完整性 (record 存在 + blob 落盘), 再动文件 — 任一缺失整体中止.
	type assetBlob struct {
		rec   asset.AssetRecord
		blobs []string
	}
	var bundleAssets []assetBlob
	for _, d := range deps {
		if d.Kind != dependency.KindTemplate && d.Kind != dependency.KindClip {
			continue
		}
		rec, ok := assetStore.Get(d.Key)
		if !ok {
			return fmt.Errorf("export: graph 引用资产 GUID %q 无 record", d.Key)
		}
		var shas []string
		for _, v := range rec.Variants {
			if !assetStore.Blobs().Has(v.Blob) {
				return fmt.Errorf("export: 资产 %q 变体 blob %s 缺失", d.Key, v.Blob)
			}
			shas = append(shas, v.Blob)
		}
		if rec.Blob != "" {
			if !assetStore.Blobs().Has(rec.Blob) {
				return fmt.Errorf("export: clip %q blob %s 缺失", d.Key, rec.Blob)
			}
			shas = append(shas, rec.Blob)
		}
		bundleAssets = append(bundleAssets, assetBlob{rec: rec, blobs: shas})
	}

	os.RemoveAll(pkgDir)
	if err := os.MkdirAll(filepath.Join(pkgDir, "embedded"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(pkgDir, "assets"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(pkgDir, "blobs"), 0o755); err != nil {
		return err
	}

	rb, _ := json.MarshalIndent(rootSg, "", "  ")
	if err := os.WriteFile(filepath.Join(pkgDir, "subgraph.json"), rb, 0o644); err != nil {
		return err
	}

	for _, d := range deps {
		if d.Kind == dependency.KindSubgraph && d.Key != rootSg.ID {
			sg, _ := readSubgraph(srcContainerRoot, d.Key)
			if sg == nil {
				continue
			}
			b, _ := json.MarshalIndent(sg, "", "  ")
			if err := os.WriteFile(filepath.Join(pkgDir, "embedded", d.Key+".json"), b, 0o644); err != nil {
				return err
			}
		}
	}

	for _, ab := range bundleAssets {
		rb, _ := json.MarshalIndent(ab.rec, "", "  ")
		if err := os.WriteFile(filepath.Join(pkgDir, "assets", ab.rec.GUID+".json"), rb, 0o644); err != nil {
			return err
		}
		for _, sha := range ab.blobs {
			data, err := assetStore.Blobs().Read(sha)
			if err != nil {
				return fmt.Errorf("export read blob %s: %w", sha, err)
			}
			if err := os.WriteFile(filepath.Join(pkgDir, "blobs", sha), data, 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

// ExportSubgraph 把容器内子图 + 递归依赖打成 library package (子图闭包根).
func (s *Store) ExportSubgraph(srcContainerRoot, rootSgID string, assetStore *asset.Store, overwrite bool) error {
	rootSg, err := readSubgraph(srcContainerRoot, rootSgID)
	if err != nil || rootSg == nil {
		return fmt.Errorf("root subgraph %q not found", rootSgID)
	}
	deps, err := dependency.ScanSubgraphDependencies(rootSgID, getNodesFromContainer(srcContainerRoot))
	if err != nil {
		return fmt.Errorf("scan deps: %w", err)
	}
	return s.writeBundle(srcContainerRoot, rootSgID, rootSg, deps, assetStore, overwrite)
}

// ExportContainer 把整容器顶层图 + 全部子图 + 资产闭包打成 library package (容器闭包根=顶层图 BFS).
// bundleID = package 目录名 (用容器 ID). 容器顶层图包成一个合成根子图存进 bundle:
// 闭包从顶层图节点 BFS, 递归 Subgraph-call → 收齐整容器的 template/clip GUID + 全子图.
func (s *Store) ExportContainer(srcContainerRoot, bundleID string, c *container.Container, assetStore *asset.Store, overwrite bool) error {
	infos := make([]dependency.NodeInfo, len(c.Graph.Nodes))
	for i, n := range c.Graph.Nodes {
		infos[i] = dependency.NodeInfo{Kind: n.Kind, Config: n.Config}
	}
	deps, err := dependency.ScanContainerDependencies(infos, getNodesFromContainer(srcContainerRoot))
	if err != nil {
		return fmt.Errorf("scan container deps: %w", err)
	}
	// 合成根子图: 用容器顶层图作 Graph, ID = bundleID. 它本身不在 deps 闭包里, 不会被当 embedded.
	rootSg := &container.Subgraph{
		ID:    bundleID,
		Label: c.Name,
		Graph: c.Graph,
	}
	return s.writeBundle(srcContainerRoot, bundleID, rootSg, deps, assetStore, overwrite)
}
