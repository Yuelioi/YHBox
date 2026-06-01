package library

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"yhbox/internal/services/container"
	"yhbox/internal/services/container/dependency"
)

// ImportConflict 单条冲突信息. Kind 是 "subgraph" | "template" | "clip".
type ImportConflict struct {
	Kind string `json:"kind"`
	Key  string `json:"key"`
}

// ImportResult Import 操作结果.
//
// MissingGlobals 反映 import 的 sg 集合 (Root+Embedded) RequiredGlobals union 减去
// 目标 container.Vars 已声明的, 是 caller 需要补 declare 的 var. dry-run (strategy="cancel") 时计算,
// FE 据此弹 "添加变量并继续" 提示 → 用 container.update RPC 补 vars 后再调真 import.
type ImportResult struct {
	Imported        []dependency.Dependency           `json:"imported"`
	Conflicts       []ImportConflict                  `json:"conflicts"`
	MissingGlobals  []container.SubgraphRequiredGlobal `json:"missingGlobals,omitempty"`
}

// ImportToContainer 把 library package 复制到容器.
// strategy: "overwrite_all" | "skip_all" | "cancel", 空串等同 overwrite_all.
//
// containerVars 是目标容器当前已声明的 vars (caller 注入, e.g. Service.ImportToContainer
// 用 LookupContainerVars 拿). 用来算 ImportResult.MissingGlobals diff. 传 nil 退化为不算 diff
// (老 caller / test fixture 兼容).
func (s *Store) ImportToContainer(libSgID, containerRoot, strategy string, containerVars []container.VarDecl) (*ImportResult, error) {
	pkg, err := s.GetSubgraphPackage(libSgID)
	if err != nil {
		return nil, err
	}

	containerSgDir := filepath.Join(containerRoot, "subgraphs")
	containerTplDir := filepath.Join(containerRoot, "templates")
	containerClipDir := filepath.Join(containerRoot, "clips")

	var conflicts []ImportConflict
	checkSg := func(sg container.Subgraph) {
		if _, err := os.Stat(filepath.Join(containerSgDir, sg.ID+".json")); err == nil {
			conflicts = append(conflicts, ImportConflict{Kind: "subgraph", Key: sg.ID})
		}
	}
	checkSg(pkg.Root)
	for _, sg := range pkg.Embedded {
		checkSg(sg)
	}
	for _, key := range pkg.Templates {
		if _, err := os.Stat(filepath.Join(containerTplDir, key+".png")); err == nil {
			conflicts = append(conflicts, ImportConflict{Kind: "template", Key: key})
		}
	}
	for _, id := range pkg.Clips {
		if _, err := os.Stat(filepath.Join(containerClipDir, id+".json")); err == nil {
			conflicts = append(conflicts, ImportConflict{Kind: "clip", Key: id})
		}
	}

	// 算 missing globals diff (无论 strategy, dry-run + 真 import 都返). FE 看 dry-run 弹 prompt.
	missingGlobals := computeMissingGlobals(pkg, containerVars)

	if len(conflicts) > 0 {
		switch strategy {
		case "cancel":
			return &ImportResult{Conflicts: conflicts, MissingGlobals: missingGlobals}, nil
		case "skip_all":
			// 后续 copy 时按 conflictKeys 跳过
		case "overwrite_all", "":
			// 默认 overwrite, 不跳过
		default:
			return nil, fmt.Errorf("unknown strategy %q", strategy)
		}
	} else if strategy == "cancel" {
		// 无 conflict + dry-run → 仍返 missing globals 让 FE 决定是否要补 var
		return &ImportResult{MissingGlobals: missingGlobals}, nil
	}
	skipAll := strategy == "skip_all"
	conflictKeys := map[string]bool{}
	for _, c := range conflicts {
		conflictKeys[c.Kind+":"+c.Key] = true
	}

	os.MkdirAll(containerSgDir, 0o755)
	os.MkdirAll(containerTplDir, 0o755)
	os.MkdirAll(containerClipDir, 0o755)

	result := &ImportResult{Conflicts: conflicts}
	writeSg := func(sg container.Subgraph) error {
		if skipAll && conflictKeys["subgraph:"+sg.ID] {
			return nil
		}
		b, _ := json.MarshalIndent(sg, "", "  ")
		if err := os.WriteFile(filepath.Join(containerSgDir, sg.ID+".json"), b, 0o644); err != nil {
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

	libPkgDir := filepath.Join(s.root, "subgraphs", libSgID)
	for _, key := range pkg.Templates {
		if skipAll && conflictKeys["template:"+key] {
			continue
		}
		if err := copyFile(filepath.Join(libPkgDir, "templates", key+".png"), filepath.Join(containerTplDir, key+".png")); err != nil {
			return nil, err
		}
		if err := copyFile(filepath.Join(libPkgDir, "templates", key+".json"), filepath.Join(containerTplDir, key+".json")); err != nil {
			return nil, err
		}
		result.Imported = append(result.Imported, dependency.Dependency{Kind: dependency.KindTemplate, Key: key})
	}
	for _, id := range pkg.Clips {
		if skipAll && conflictKeys["clip:"+id] {
			continue
		}
		srcJSON := filepath.Join(libPkgDir, "clips", id+".json")
		if err := copyFile(srcJSON, filepath.Join(containerClipDir, id+".json")); err != nil {
			return nil, err
		}
		srcCsbf := filepath.Join(libPkgDir, "clips", id+".csbf")
		if _, err := os.Stat(srcCsbf); err == nil {
			copyFile(srcCsbf, filepath.Join(containerClipDir, id+".csbf"))
		}
		result.Imported = append(result.Imported, dependency.Dependency{Kind: dependency.KindClip, Key: id})
	}
	result.MissingGlobals = missingGlobals
	return result, nil
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

// ExportFromContainer 把容器内子图 + 递归依赖打成 library package.
// scanDeps: 调用方注入 dependency.ScanSubgraphDependencies (避免 library 直接依赖 dependency.GetExtractor).
func (s *Store) ExportFromContainer(
	srcContainerRoot string,
	rootSgID string,
	scanDeps func(rootSgID string, getNodes func(string) ([]dependency.NodeInfo, error)) ([]dependency.Dependency, error),
	overwrite bool,
) error {
	readSubgraph := func(sgID string) (*container.Subgraph, error) {
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
	getNodes := func(sgID string) ([]dependency.NodeInfo, error) {
		sg, err := readSubgraph(sgID)
		if err != nil || sg == nil {
			return nil, err
		}
		nodes := make([]dependency.NodeInfo, len(sg.Graph.Nodes))
		for i, n := range sg.Graph.Nodes {
			nodes[i] = dependency.NodeInfo{Kind: n.Kind, Config: n.Config}
		}
		return nodes, nil
	}

	deps, err := scanDeps(rootSgID, getNodes)
	if err != nil {
		return fmt.Errorf("scan deps: %w", err)
	}

	pkgDir := filepath.Join(s.root, "subgraphs", rootSgID)
	if !overwrite {
		if _, err := os.Stat(pkgDir); err == nil {
			return fmt.Errorf("library package %q already exists, use overwrite=true to replace", rootSgID)
		}
	}
	os.RemoveAll(pkgDir)
	os.MkdirAll(filepath.Join(pkgDir, "embedded"), 0o755)
	os.MkdirAll(filepath.Join(pkgDir, "templates"), 0o755)
	os.MkdirAll(filepath.Join(pkgDir, "clips"), 0o755)

	rootSg, err := readSubgraph(rootSgID)
	if err != nil || rootSg == nil {
		return fmt.Errorf("root subgraph %q not found", rootSgID)
	}
	rb, _ := json.MarshalIndent(rootSg, "", "  ")
	if err := os.WriteFile(filepath.Join(pkgDir, "subgraph.json"), rb, 0o644); err != nil {
		return err
	}

	for _, d := range deps {
		switch d.Kind {
		case dependency.KindSubgraph:
			if d.Key == rootSgID {
				continue
			}
			sg, _ := readSubgraph(d.Key)
			if sg == nil {
				continue
			}
			b, _ := json.MarshalIndent(sg, "", "  ")
			os.WriteFile(filepath.Join(pkgDir, "embedded", d.Key+".json"), b, 0o644)
		case dependency.KindTemplate:
			copyFile(filepath.Join(srcContainerRoot, "templates", d.Key+".png"), filepath.Join(pkgDir, "templates", d.Key+".png"))
			copyFile(filepath.Join(srcContainerRoot, "templates", d.Key+".json"), filepath.Join(pkgDir, "templates", d.Key+".json"))
		case dependency.KindClip:
			copyFile(filepath.Join(srcContainerRoot, "clips", d.Key+".json"), filepath.Join(pkgDir, "clips", d.Key+".json"))
			srcCsbf := filepath.Join(srcContainerRoot, "clips", d.Key+".csbf")
			if _, err := os.Stat(srcCsbf); err == nil {
				copyFile(srcCsbf, filepath.Join(pkgDir, "clips", d.Key+".csbf"))
			}
		}
	}
	return nil
}
