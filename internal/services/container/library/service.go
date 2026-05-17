package library

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"yhbox/internal/services/container"
)

// containerStoreAccess library 包不能直接 import container 主包的 Service 接口（避免循环）。
// 用窄接口注入：仅 Get + SaveSubgraph 两个方法。
type containerStoreAccess interface {
	Get(id string) (container.Container, bool)
	SaveSubgraph(containerID string, sg *container.Subgraph) error
}

// Service 给 wails3 RPC 用的瘦皮层。所有方法直接转发到 Store。
type Service struct {
	store                *Store
	emit                 func(name string, data any)
	containerStoreAccess containerStoreAccess
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

func (s *Service) ListSubgraphs() []LibrarySubgraph {
	return s.store.ListSubgraphs()
}

func (s *Service) GetSubgraph(id string) (LibrarySubgraph, error) {
	sg, ok := s.store.GetSubgraph(id)
	if !ok {
		return LibrarySubgraph{}, fmt.Errorf("library subgraph %q not found", id)
	}
	return sg, nil
}

func (s *Service) DeleteSubgraph(id string) error {
	if err := s.store.DeleteSubgraph(id); err != nil {
		return err
	}
	s.emitChanged()
	return nil
}

func (s *Service) DeleteSubgraphMany(ids []string) error {
	for _, id := range ids {
		if err := s.store.DeleteSubgraph(id); err != nil {
			return fmt.Errorf("delete %q: %w", id, err)
		}
	}
	s.emitChanged()
	return nil
}

func (s *Service) ListTemplates() []TemplateEntry {
	return s.store.ListTemplates()
}

// SetContainerStoreAccess main.go 启动期注入。
func (s *Service) SetContainerStoreAccess(c containerStoreAccess) {
	s.containerStoreAccess = c
}

// CopyToContainer copy-on-use：把库 subgraph 克隆为目标容器内的新独立子图。
func (s *Service) CopyToContainer(libSubgraphID, containerID string) (container.Subgraph, error) {
	src, ok := s.store.GetSubgraph(libSubgraphID)
	if !ok {
		return container.Subgraph{}, fmt.Errorf("library subgraph %q not found", libSubgraphID)
	}
	if s.containerStoreAccess == nil {
		return container.Subgraph{}, fmt.Errorf("container store access not injected")
	}
	targetCont, ok := s.containerStoreAccess.Get(containerID)
	if !ok {
		return container.Subgraph{}, fmt.Errorf("container %q not found", containerID)
	}
	// v1 简化：existingKeys 暂传空集（不做 template key 冲突检测）
	existingKeys := map[string]struct{}{}

	result, err := CopySubgraph(&src, &targetCont, existingKeys, nil)
	if err != nil {
		return container.Subgraph{}, err
	}
	if err := s.containerStoreAccess.SaveSubgraph(containerID, result.NewSubgraph); err != nil {
		return container.Subgraph{}, err
	}
	s.emitChanged()
	return *result.NewSubgraph, nil
}

// PublishFromContainer 把容器内的子图发布到库 (容器→库, 反向于 CopyToContainer).
// 深拷贝 + 重发 sg.ID / graph.ID / OutputPins decl ID (内部 SubgraphOutput.config.declID
// 同步改写). 容器里原 sg 不动. v1 暂不复制 referenced templates,
// 用户需保证库里已有对应 template key.
func (s *Service) PublishFromContainer(containerID, sgID string) (LibrarySubgraph, error) {
	if s.containerStoreAccess == nil {
		return LibrarySubgraph{}, fmt.Errorf("container store access not injected")
	}
	cont, ok := s.containerStoreAccess.Get(containerID)
	if !ok {
		return LibrarySubgraph{}, fmt.Errorf("container %q not found", containerID)
	}
	var src *container.Subgraph
	for i := range cont.Subgraphs {
		if cont.Subgraphs[i].ID == sgID {
			src = &cont.Subgraphs[i]
			break
		}
	}
	if src == nil {
		return LibrarySubgraph{}, fmt.Errorf("subgraph %q not found in container %q", sgID, containerID)
	}

	// 深拷贝走 json round-trip, 干净
	b, err := json.Marshal(*src)
	if err != nil {
		return LibrarySubgraph{}, fmt.Errorf("marshal src: %w", err)
	}
	var cp container.Subgraph
	if err := json.Unmarshal(b, &cp); err != nil {
		return LibrarySubgraph{}, fmt.Errorf("unmarshal cp: %w", err)
	}

	// 重发 ID
	cp.ID = "sg-" + uuid.NewString()[:8]
	cp.Graph.ID = uuid.NewString()
	declIDMap := map[string]string{}
	newPins := make([]container.SubgraphOutputDecl, len(cp.OutputPins))
	for i, p := range cp.OutputPins {
		newID := uuid.NewString()
		declIDMap[p.ID] = newID
		newPins[i] = container.SubgraphOutputDecl{ID: newID, Name: p.Name}
	}
	cp.OutputPins = newPins
	// 同步改写内部 SubgraphOutput.config.declID
	for i := range cp.Graph.Nodes {
		n := &cp.Graph.Nodes[i]
		if n.Kind != "SubgraphOutput" || n.Config == nil {
			continue
		}
		old, _ := n.Config["declID"].(string)
		if newD, ok := declIDMap[old]; ok {
			n.Config["declID"] = newD
		}
	}

	libSg := LibrarySubgraph{Subgraph: cp}
	if err := s.store.SaveSubgraph(&libSg); err != nil {
		return LibrarySubgraph{}, err
	}
	s.emitChanged()
	return libSg, nil
}

// DeleteTemplateMany 批量删除库 template。
func (s *Service) DeleteTemplateMany(keys []string) error {
	var errs []string
	for _, k := range keys {
		if err := s.store.DeleteTemplate(k); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", k, err))
		}
	}
	s.emitChanged()
	if len(errs) > 0 {
		return fmt.Errorf("部分失败: %s", strings.Join(errs, "; "))
	}
	return nil
}
