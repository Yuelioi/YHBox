package library

import (
	"fmt"
	"path/filepath"

	"yotta/internal/services/asset"
	"yotta/internal/services/container"
)

// Service 给 wails3 RPC 用的瘦皮层。所有方法直接转发到 Store。
type Service struct {
	store          *Store
	assetStore     *asset.Store
	emit           func(name string, data any)
	containersRoot string
	// 注入回调拿目标容器当前 Vars[], 给 import 算 MissingGlobals diff 用.
	// nil 时退化为不算 diff. main.go SetContainerLookup(containerStore.Get).
	lookupContainer func(id string) (container.Container, bool)
	// 注入回调让 import 写盘后同步容器 Store 内存缓存 (containerStore.Reload).
	// import 直接写 subgraphs/*.json 绕过 Store, 不 reload 则 ListSubgraphs 返旧值,
	// 刚导入的子图查不到 → 节点 "(子图未找到)". nil 时跳过 (测试可省).
	reloadContainer func(id string) error
}

func NewService(store *Store, assetStore *asset.Store) *Service {
	return &Service{store: store, assetStore: assetStore}
}

func (s *Service) SetEmit(emit func(name string, data any)) { s.emit = emit }

// SetContainersRoot 注入容器数据根目录 (e.g. dataDir/containers).
// ImportToContainer + ExportSubgraph 用来把 containerID 解析成文件系统路径.
func (s *Service) SetContainersRoot(dir string) { s.containersRoot = dir }

// SetContainerLookup 注入 containerStore.Get 让 ImportToContainer 算 MissingGlobals diff.
func (s *Service) SetContainerLookup(f func(id string) (container.Container, bool)) {
	s.lookupContainer = f
}

// SetContainerReloader 注入 containerStore.Reload 让 ImportToContainer 写盘后刷新目标容器内存缓存.
func (s *Service) SetContainerReloader(f func(id string) error) {
	s.reloadContainer = f
}

func (s *Service) emitChanged() {
	if s.emit != nil {
		s.emit("library:changed", map[string]any{})
	}
}

func (s *Service) containerRoot(containerID string) (string, error) {
	if s.containersRoot == "" {
		return "", fmt.Errorf("containersRoot 未注入 (main.go SetContainersRoot 调了吗?)")
	}
	return filepath.Join(s.containersRoot, containerID), nil
}

func (s *Service) ListSubgraphs() ([]string, error) {
	return s.store.ListSubgraphs()
}

func (s *Service) GetSubgraphPackage(sgID string) (*SubgraphPackage, error) {
	return s.store.GetSubgraphPackage(sgID)
}

func (s *Service) DeleteSubgraphPackage(sgID string) error {
	if err := s.store.DeleteSubgraphPackage(sgID); err != nil {
		return err
	}
	s.emitChanged()
	return nil
}

// ImportToContainer 把 library package 幂等合并到容器 + 全局 asset 库.
// containerID 是目标容器 ID (由 containersRoot 解析为文件系统路径).
//
// 资产按 GUID/sha 幂等合并 (加性变体), 图写入为提交点, 无 conflict/strategy.
// 用 lookupContainer 拿目标容器 Vars 算 MissingGlobals diff (lookup nil 时跳过 diff).
func (s *Service) ImportToContainer(libSgID, containerID string) (*ImportResult, error) {
	root, err := s.containerRoot(containerID)
	if err != nil {
		return nil, err
	}
	var containerVars []container.VarDecl
	if s.lookupContainer != nil {
		if c, ok := s.lookupContainer(containerID); ok {
			containerVars = c.Vars
		}
	}
	result, err := s.store.ImportToContainer(libSgID, root, s.assetStore, containerVars)
	if err != nil {
		return nil, err
	}
	// import 直接写盘绕过容器 Store, reload 让内存缓存看见新子图 (否则 ListSubgraphs 返旧值).
	if s.reloadContainer != nil {
		if err := s.reloadContainer(containerID); err != nil {
			return nil, fmt.Errorf("reload container %q after import: %w", containerID, err)
		}
	}
	s.emitChanged()
	return result, nil
}

// ExportSubgraph 把容器内子图 + 递归依赖 + 资产闭包打成 library package.
func (s *Service) ExportSubgraph(containerID, sgID string, overwrite bool) error {
	root, err := s.containerRoot(containerID)
	if err != nil {
		return err
	}
	if err := s.store.ExportSubgraph(root, sgID, s.assetStore, overwrite); err != nil {
		return err
	}
	s.emitChanged()
	return nil
}

// ExportContainer 把整容器顶层图 + 全部子图 + 资产闭包打成 library package (bundle ID = 容器 ID).
func (s *Service) ExportContainer(containerID string, overwrite bool) error {
	root, err := s.containerRoot(containerID)
	if err != nil {
		return err
	}
	if s.lookupContainer == nil {
		return fmt.Errorf("lookupContainer 未注入 (main.go SetContainerLookup 调了吗?)")
	}
	c, ok := s.lookupContainer(containerID)
	if !ok {
		return fmt.Errorf("container %q not found", containerID)
	}
	if err := s.store.ExportContainer(root, containerID, &c, s.assetStore, overwrite); err != nil {
		return err
	}
	s.emitChanged()
	return nil
}
