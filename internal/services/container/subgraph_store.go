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

// normalizeSubgraph self-heal + 老格式 marker 迁移:
//   - 扫 sg.Graph.Nodes 里老格式 SubgraphInput/Output 节点, 提取到 sg.Entry / sg.OutputPins[].NodeID metadata, 节点本身删.
//   - 没 Entry → 生成新 entry NodeID + 默认位置.
//   - 没 OutputPins[i].NodeID → 生成新 out NodeID + 错位排开.
//   - 空 OutputPins → 兜底加一个 "done" pin.
// 幂等: 已 normalize 过的 sg 再调一次, sg.Entry.NodeID / OutputPins[*].NodeID 都存在, 不动节点列表.
func normalizeSubgraph(sg *Subgraph) {
	// 迁移老格式 marker 节点 → metadata.
	// 暂存找到的 SubgraphOutput 节点, 后面统一 reconcile (避免 declID 缺失/未声明 OutputPin 时丢节点 ID).
	type outputNode struct {
		nodeID string
		declID string
		x, y   float32
	}
	var foundOutputs []outputNode
	nextNodes := sg.Graph.Nodes[:0]
	for _, n := range sg.Graph.Nodes {
		switch n.Kind {
		case "SubgraphInput":
			if sg.Entry.NodeID == "" {
				sg.Entry = SubgraphMarker{NodeID: n.ID, X: n.X, Y: n.Y}
			}
			// strip
		case "SubgraphOutput":
			declID, _ := n.Config["DeclID"].(string)
			foundOutputs = append(foundOutputs, outputNode{nodeID: n.ID, declID: declID, x: n.X, y: n.Y})
			// strip
		default:
			nextNodes = append(nextNodes, n)
		}
	}
	sg.Graph.Nodes = nextNodes

	// Reconcile SubgraphOutput 节点 → OutputPins[].NodeID:
	//   1. declID 匹配某 OutputPin → 填它的 NodeID/X/Y (若空)
	//   2. declID 非空且不匹配 → 自动 append 新 OutputPin (老 bug self-heal)
	//   3. declID 空 → 找第一个无 NodeID 的 OutputPin 填, 没空位则 append 新 (Name="done")
	for _, on := range foundOutputs {
		assigned := false
		if on.declID != "" {
			for i := range sg.OutputPins {
				if sg.OutputPins[i].ID == on.declID {
					if sg.OutputPins[i].NodeID == "" {
						sg.OutputPins[i].NodeID = on.nodeID
						sg.OutputPins[i].X = on.x
						sg.OutputPins[i].Y = on.y
					}
					assigned = true
					break
				}
			}
			if !assigned {
				sg.OutputPins = append(sg.OutputPins, SubgraphOutputDecl{
					ID: on.declID, Name: "done", NodeID: on.nodeID, X: on.x, Y: on.y,
				})
				assigned = true
			}
		}
		if !assigned {
			// declID 空 — 找第一个无 NodeID 的 OutputPin 填; 没空位 → append 新 (genID)
			for i := range sg.OutputPins {
				if sg.OutputPins[i].NodeID == "" {
					sg.OutputPins[i].NodeID = on.nodeID
					sg.OutputPins[i].X = on.x
					sg.OutputPins[i].Y = on.y
					assigned = true
					break
				}
			}
			if !assigned {
				sg.OutputPins = append(sg.OutputPins, SubgraphOutputDecl{
					ID: uuid.NewString(), Name: "done", NodeID: on.nodeID, X: on.x, Y: on.y,
				})
			}
		}
	}

	// 补齐 Entry.
	if sg.Entry.NodeID == "" {
		sg.Entry = SubgraphMarker{NodeID: "entry-" + uuid.NewString()[:8], X: 80, Y: 160}
	}
	// 兜底 OutputPins 非空.
	if len(sg.OutputPins) == 0 {
		sg.OutputPins = []SubgraphOutputDecl{{ID: uuid.NewString(), Name: "done"}}
	}
	// 补齐每个 OutputPin 的 NodeID + 默认位置.
	for i := range sg.OutputPins {
		if sg.OutputPins[i].NodeID == "" {
			sg.OutputPins[i].NodeID = "out-" + uuid.NewString()[:8]
			sg.OutputPins[i].X = 420
			sg.OutputPins[i].Y = 160 + float32(i)*80
		}
	}
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
	// 单独保 sg 也走 self-heal (跟 store.load / Container.Normalize 同一入口).
	normalizeSubgraph(sg)
	// 同步 RequiredGlobals — 从 in-memory container.Vars 拿 type/default.
	sg.RequiredGlobals = computeRequiredGlobals(sg, c.Vars)
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
