// service_subgraph.go — 全局子图池的 wails3 RPC 皮层 (2026-06-12 全局化).
// 签名无 containerID; 保存/删除带基准 rev 乐观锁; 删除安全靠 Referrers 查询 (FE 弹
// 「被 N 处引用」警告, 同模板删除现状).
package container

import (
	"encoding/json"
	"fmt"
	"time"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/google/uuid"
)

// SubgraphReferrer 一处引用某子图的节点位置 (形态同 asset.Referrer, 不跨包复用避免反向依赖).
type SubgraphReferrer struct {
	ContainerID string `json:"containerID"`
	SubgraphID  string `json:"subgraphID,omitempty"` // 引用方所在子图; 空 = 容器顶层图
	NodeID      string `json:"nodeID"`
	NodeKind    string `json:"nodeKind"`
}

// SubgraphService wails3 RPC 入口.
type SubgraphService struct {
	store *SubgraphStore
	emit  func(name string, data any)
	// scanReferrers 反向引用扫描 (wire 层注入 — 那里有容器 store 访问权).
	// 全容器图 + 全池子图图体, 内存遍历. nil → Referrers 返空 (测试语境).
	scanReferrers func(sgID string) []SubgraphReferrer
}

func NewSubgraphService(store *SubgraphStore) *SubgraphService {
	return &SubgraphService{store: store}
}

func (s *SubgraphService) SetEmit(emit func(name string, data any)) { s.emit = emit }

// SetReferrerScanner 注入反向引用扫描 (main.go wire).
func (s *SubgraphService) SetReferrerScanner(f func(sgID string) []SubgraphReferrer) {
	s.scanReferrers = f
}

func (s *SubgraphService) emitChanged() {
	if s.emit != nil {
		s.emit("subgraph:changed", map[string]any{})
	}
}

// List 全池快照 (含匿名 — 匿名隐藏是 FE 库浏览层的事).
func (s *SubgraphService) List() []Subgraph {
	return s.store.List()
}

func (s *SubgraphService) Get(id string) (Subgraph, error) {
	sg, ok := s.store.Get(id)
	if !ok {
		return Subgraph{}, fmt.Errorf("subgraph %q not found", id)
	}
	return sg, nil
}

// Create 新建子图 (入全局池), 自动铸 sg-<完整uuid>。
// 预填 SubgraphInput + SubgraphOutput 节点：validator 要求每个 sg 必须有 ≥1 SubgraphOutput，
// 且 SubgraphOutput.config.declID 必须挂在 OutputPins 的某个 ID 上（runtime 才能 dispatch 父图）。
// 不预填 → 用户每建一个子图都会撞 EMPTY_SUBGRAPH_OUTPUT。
func (s *SubgraphService) Create(label string) (Subgraph, error) {
	declID := uuid.NewString()
	now := time.Now().UTC()
	sg := Subgraph{
		Label: label,
		Graph: Graph{
			ID:            uuid.NewString(),
			SchemaVersion: GraphSchemaVersion,
			Nodes: []GraphNode{
				{ID: "in", Kind: "SubgraphInput", X: 80, Y: 160, CreatedAt: now},
				{ID: "out", Kind: "SubgraphOutput", X: 420, Y: 160,
					Config: map[string]any{"DeclID": declID}, CreatedAt: now},
			},
		},
		OutputPins: []SubgraphOutputDecl{{ID: declID, Name: "done"}},
		CreatedAt:  now,
	}
	if err := s.store.Create(&sg); err != nil {
		return Subgraph{}, err
	}
	s.emitChanged()
	return sg, nil
}

// Update 接 RFC7386 merge patch + 乐观锁基准 rev (盘上更新则拒, FE 提示重载).
func (s *SubgraphService) Update(id, patchJSON string, baseRev int64) error {
	cur, ok := s.store.Get(id)
	if !ok {
		return fmt.Errorf("subgraph %q not found", id)
	}
	curJSON, err := json.Marshal(cur)
	if err != nil {
		return err
	}
	merged, err := jsonpatch.MergePatch(curJSON, []byte(patchJSON))
	if err != nil {
		return fmt.Errorf("merge patch: %w", err)
	}
	var next Subgraph
	if err := json.Unmarshal(merged, &next); err != nil {
		return err
	}
	next.ID = id // 防 patch 改 id (ID 终身不变)
	if err := s.store.Update(&next, baseRev); err != nil {
		return err
	}
	s.emitChanged()
	return nil
}

// Delete 删除 + 乐观锁. 删除安全: FE 先调 Referrers 弹「被引用」警告, 用户确认才调这里.
func (s *SubgraphService) Delete(id string, baseRev int64) error {
	if err := s.store.Delete(id, baseRev); err != nil {
		return err
	}
	s.emitChanged()
	return nil
}

// Duplicate 复制为新子图 (fork 入口) — 新 ID / rev=1 / 一律具名.
func (s *SubgraphService) Duplicate(id string) (Subgraph, error) {
	dup, err := s.store.Duplicate(id)
	if err != nil {
		return Subgraph{}, err
	}
	s.emitChanged()
	return dup, nil
}

// Referrers 谁在引用此子图 — FE 删除前警告 + 库页「被 N 个容器使用」(按 ContainerID 去重).
func (s *SubgraphService) Referrers(id string) []SubgraphReferrer {
	if s.scanReferrers == nil {
		return nil
	}
	return s.scanReferrers(id)
}
