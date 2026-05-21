package library

import (
	"yhbox/internal/services/container/dependency"
)

// Service 给 wails3 RPC 用的瘦皮层。所有方法直接转发到 Store。
type Service struct {
	store *Store
	emit  func(name string, data any)
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func (s *Service) SetEmit(emit func(name string, data any)) { s.emit = emit }

func (s *Service) emitChanged() {
	if s.emit != nil {
		s.emit("library:changed", map[string]any{})
	}
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

// ImportToContainer 把 library package 复制到容器目录.
// containerRoot 是容器的根目录 (e.g. dataDir/containers/<cid>).
func (s *Service) ImportToContainer(libSgID, containerRoot, strategy string) (*ImportResult, error) {
	result, err := s.store.ImportToContainer(libSgID, containerRoot, strategy)
	if err != nil {
		return nil, err
	}
	s.emitChanged()
	return result, nil
}

// ExportFromContainer 把容器内子图 + 递归依赖打成 library package.
func (s *Service) ExportFromContainer(
	srcContainerRoot string,
	rootSgID string,
	scanDeps func(rootSgID string, getNodes func(string) ([]dependency.NodeInfo, error)) ([]dependency.Dependency, error),
	overwrite bool,
) error {
	if err := s.store.ExportFromContainer(srcContainerRoot, rootSgID, scanDeps, overwrite); err != nil {
		return err
	}
	s.emitChanged()
	return nil
}
