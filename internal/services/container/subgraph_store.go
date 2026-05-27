package container

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// normalizeSubgraph self-heal 旧版 CreateSubgraph 漏建 SubgraphOutput 节点的子图.
// validator 要求每个 sg 必须有 ≥1 SubgraphOutput; 早期阶段持久化数据可能缺失,
// 读盘后自动补一个 + 绑定 OutputPins[0].ID 当 declID. 下次 save 写回的就是 normalized 版本.
// 已有 SubgraphOutput 的子图不动.
func normalizeSubgraph(sg *Subgraph) {
	for _, n := range sg.Graph.Nodes {
		if n.Kind == "SubgraphOutput" {
			return
		}
	}
	if len(sg.OutputPins) == 0 {
		sg.OutputPins = []SubgraphOutputDecl{{ID: uuid.NewString(), Name: "done"}}
	}
	declID := sg.OutputPins[0].ID
	sg.Graph.Nodes = append(sg.Graph.Nodes, GraphNode{
		ID: "out-" + uuid.NewString()[:8], Kind: "SubgraphOutput",
		X: 420, Y: 160,
		Config:    map[string]any{"DeclID": declID},
		CreatedAt: time.Now().UTC(),
	})
}

// SaveSubgraph 写一个 subgraph 到 containers/<cid>/subgraphs/<sg.ID>.json，
// 并同步更新内存里的 container.Subgraphs。
func (s *Store) SaveSubgraph(containerID string, sg *Subgraph) error {
	if err := validateID(sg.ID); err != nil {
		return fmt.Errorf("subgraph id: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.byID[containerID]
	if !ok {
		return fmt.Errorf("container %q not found", containerID)
	}
	if c.Status == StatusIncompatible {
		return fmt.Errorf("container %q is incompatible, refuse to modify", containerID)
	}

	now := time.Now().UTC()
	if sg.CreatedAt.IsZero() {
		sg.CreatedAt = now
	}
	// B3: 单独保 sg 也走 self-heal — 之前只 store.load() / Container.Normalize 调, SaveSubgraph 漏.
	normalizeSubgraph(sg)
	dir := filepath.Join(s.root, containerID, "subgraphs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, sg.ID+".json")
	tmp := path + ".tmp"
	b, err := json.MarshalIndent(sg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	// 同步内存
	replaced := false
	for i := range c.Subgraphs {
		if c.Subgraphs[i].ID == sg.ID {
			c.Subgraphs[i] = *sg
			replaced = true
			break
		}
	}
	if !replaced {
		c.Subgraphs = append(c.Subgraphs, *sg)
	}
	s.byID[containerID] = c
	return nil
}

// GetSubgraph 取容器内的指定子图。
func (s *Store) GetSubgraph(containerID, sgID string) (Subgraph, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.byID[containerID]
	if !ok {
		return Subgraph{}, false
	}
	for _, sg := range c.Subgraphs {
		if sg.ID == sgID {
			return sg, true
		}
	}
	return Subgraph{}, false
}

// ListSubgraphs 列出某容器全部子图（拷贝防 mutation）。
func (s *Store) ListSubgraphs(containerID string) []Subgraph {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.byID[containerID]
	if !ok {
		return nil
	}
	out := make([]Subgraph, len(c.Subgraphs))
	copy(out, c.Subgraphs)
	return out
}

// DeleteSubgraph 删 containers/<cid>/subgraphs/<sg>.json 并从内存里移除。
// 不存在视为成功（idempotent）。
func (s *Store) DeleteSubgraph(containerID, sgID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.byID[containerID]
	if !ok {
		return fmt.Errorf("container %q not found", containerID)
	}
	path := filepath.Join(s.root, containerID, "subgraphs", sgID+".json")
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	next := c.Subgraphs[:0]
	for _, sg := range c.Subgraphs {
		if sg.ID != sgID {
			next = append(next, sg)
		}
	}
	c.Subgraphs = next
	s.byID[containerID] = c
	return nil
}
