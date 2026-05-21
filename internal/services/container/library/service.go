package library

import (
	"fmt"
	"path/filepath"

	"yhbox/internal/services/container/dependency"
)

// Service 给 wails3 RPC 用的瘦皮层。所有方法直接转发到 Store。
type Service struct {
	store          *Store
	emit           func(name string, data any)
	containersRoot string
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func (s *Service) SetEmit(emit func(name string, data any)) { s.emit = emit }

// SetContainersRoot 注入容器数据根目录 (e.g. dataDir/containers).
// ImportToContainer + ExportSubgraph 用来把 containerID 解析成文件系统路径.
func (s *Service) SetContainersRoot(dir string) { s.containersRoot = dir }

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

// ImportToContainer 把 library package 复制到容器.
// containerID 是目标容器 ID (由 containersRoot 解析为文件系统路径).
func (s *Service) ImportToContainer(libSgID, containerID, strategy string) (*ImportResult, error) {
	root, err := s.containerRoot(containerID)
	if err != nil {
		return nil, err
	}
	result, err := s.store.ImportToContainer(libSgID, root, strategy)
	if err != nil {
		return nil, err
	}
	s.emitChanged()
	return result, nil
}

// ExportSubgraph 把容器内子图 + 递归依赖打成 library package.
func (s *Service) ExportSubgraph(containerID, sgID string, overwrite bool) error {
	root, err := s.containerRoot(containerID)
	if err != nil {
		return err
	}
	scanDeps := func(rootSgID string, getNodes func(string) ([]dependency.NodeInfo, error)) ([]dependency.Dependency, error) {
		return dependency.ScanSubgraphDependencies(rootSgID, getNodes, dependency.GetExtractor)
	}
	if err := s.store.ExportFromContainer(root, sgID, scanDeps, overwrite); err != nil {
		return err
	}
	s.emitChanged()
	return nil
}
