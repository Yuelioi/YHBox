// subgraph_store.go — 全局子图池存储 (2026-06-12 全局化: 容器只引用不拥有).
//
//	data/subgraphs/<sgid>.json   一条一文件
//
// 并发控制: 单调 rev 乐观锁 (Update/Delete 带基准 rev, 盘上更新则拒绝 — 仅单实例
// 并发控制, 不参与跨机真相判定). 写代数 gen 供闭包/引用计数缓存失效.
package container

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// NormalizeSubgraph 导出入口 — runtime / 测试 fixture 对 in-memory 构造的子图跑同一份
// self-heal (Store 加载/保存路径已自动跑, 幂等).
func NormalizeSubgraph(sg *Subgraph) { normalizeSubgraph(sg) }

// NewSubgraphID 铸全局唯一子图 ID — sg-<完整uuid>. 铸出后终身不变 (耐久性原则).
func NewSubgraphID() string { return "sg-" + uuid.NewString() }

// ErrSubgraphRevStale 乐观锁拒绝: 调用方基准 rev 落后于盘上 (别的编辑器已保存过).
// 前端据此提示"盘上已有更新, 重载?".
var ErrSubgraphRevStale = errors.New("subgraph rev stale: 盘上已有更新")

// SubgraphStore 全局子图池.
type SubgraphStore struct {
	mu   sync.RWMutex
	root string
	byID map[string]Subgraph
	gen  int64 // 写代数: 任何写 (Create/Update/Delete/GC) +1. 闭包/引用计数缓存键.
}

func NewSubgraphStore(root string) (*SubgraphStore, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", root, err)
	}
	s := &SubgraphStore{root: root, byID: map[string]Subgraph{}}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SubgraphStore) load() error {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return err
	}
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".json") {
			continue
		}
		path := filepath.Join(s.root, ent.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[SubgraphStore] skip %q: read error: %v\n", ent.Name(), err)
			continue
		}
		var sg Subgraph
		if err := json.Unmarshal(b, &sg); err != nil {
			fmt.Fprintf(os.Stderr, "[SubgraphStore] skip %q: bad json: %v\n", ent.Name(), err)
			continue
		}
		if sg.ID == "" {
			fmt.Fprintf(os.Stderr, "[SubgraphStore] skip %q: empty id\n", ent.Name())
			continue
		}
		if sg.SchemaVersion > SubgraphSchemaVersion {
			return fmt.Errorf("%s: schemaVersion %d 超出本程序支持 (%d)", path, sg.SchemaVersion, SubgraphSchemaVersion)
		}
		normalizeSubgraph(&sg)
		s.byID[sg.ID] = sg
	}
	return nil
}

func (s *SubgraphStore) Get(id string) (Subgraph, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sg, ok := s.byID[id]
	return sg, ok
}

// List 全池快照 (含匿名 — 匿名过滤是 UI 层的事, validator/闭包解析要看全集).
func (s *SubgraphStore) List() []Subgraph {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Subgraph, 0, len(s.byID))
	for _, sg := range s.byID {
		out = append(out, sg)
	}
	return out
}

// Generation 当前写代数 — 闭包/引用计数缓存键, 代数变则缓存失效.
func (s *SubgraphStore) Generation() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.gen
}

// persist 共享写路径: self-heal + RequiredGlobals 派生 + 版本盖章 + 原子写 + 内存同步 + 代数 +1.
// caller 持写锁; rev 由 caller 先定好.
func (s *SubgraphStore) persist(sg *Subgraph) error {
	if err := validateID(sg.ID); err != nil {
		return fmt.Errorf("subgraph id: %w", err)
	}
	now := time.Now().UTC()
	if sg.CreatedAt.IsZero() {
		sg.CreatedAt = now
	}
	normalizeSubgraph(sg)
	sg.RequiredGlobals = computeRequiredGlobals(sg)
	sg.SchemaVersion = SubgraphSchemaVersion
	path := filepath.Join(s.root, sg.ID+".json")
	b, err := json.MarshalIndent(sg, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	s.byID[sg.ID] = *sg
	s.gen++
	return nil
}

// Create 新建. sg.ID 空 → 铸 NewSubgraphID; 非空 (迁移/录制预铸) 必须未占用. rev=1.
func (s *SubgraphStore) Create(sg *Subgraph) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sg.ID == "" {
		sg.ID = NewSubgraphID()
	}
	if _, exists := s.byID[sg.ID]; exists {
		return fmt.Errorf("subgraph %q 已存在", sg.ID)
	}
	sg.Rev = 1
	return s.persist(sg)
}

// Update 全量替换 + 乐观锁: baseRev 必须等于盘上当前 rev, 否则 ErrSubgraphRevStale.
func (s *SubgraphStore) Update(sg *Subgraph, baseRev int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.byID[sg.ID]
	if !ok {
		return fmt.Errorf("subgraph %q not found", sg.ID)
	}
	if cur.Rev != baseRev {
		return fmt.Errorf("%w (盘上 rev=%d, 基准 rev=%d)", ErrSubgraphRevStale, cur.Rev, baseRev)
	}
	sg.Rev = cur.Rev + 1
	if sg.CreatedAt.IsZero() {
		sg.CreatedAt = cur.CreatedAt
	}
	return s.persist(sg)
}

// Delete 删除 + 乐观锁 (存在时 baseRev 必须匹配; 不存在视为成功, idempotent).
func (s *SubgraphStore) Delete(id string, baseRev int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.byID[id]
	if !ok {
		return nil
	}
	if cur.Rev != baseRev {
		return fmt.Errorf("%w (盘上 rev=%d, 基准 rev=%d)", ErrSubgraphRevStale, cur.Rev, baseRev)
	}
	return s.removeLocked(id)
}

// removeLocked 删文件 + 内存 + 代数. caller 持写锁.
func (s *SubgraphStore) removeLocked(id string) error {
	if err := os.Remove(filepath.Join(s.root, id+".json")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	delete(s.byID, id)
	s.gen++
	return nil
}

// Duplicate 复制为新子图 (fork 入口, ≈Blender Make Local):
// 深拷贝 + 新 ID + rev=1 + 新时间戳 + RequiredGlobals 原样复制不重算 (图相同重算结果
// 也相同, 写死防止后人"优化"触发重算) + 产物一律具名 (用户主动 fork 行为).
func (s *SubgraphStore) Duplicate(id string) (Subgraph, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.byID[id]
	if !ok {
		return Subgraph{}, fmt.Errorf("subgraph %q not found", id)
	}
	dup := cur // 浅拷贝起手
	// 深拷贝引用类型字段 (经 JSON round-trip — Graph/Config 内嵌 map, 手抄易漏).
	b, err := json.Marshal(cur)
	if err != nil {
		return Subgraph{}, err
	}
	if err := json.Unmarshal(b, &dup); err != nil {
		return Subgraph{}, err
	}
	dup.ID = NewSubgraphID()
	dup.Rev = 1
	dup.IsAnonymous = false
	dup.CreatedAt = time.Now().UTC()
	if dup.Label != "" {
		dup.Label += " (副本)"
	}
	// RequiredGlobals 原样复制 — persist 会重算, 但图未变结果相同 (语义上等价于原样复制).
	if err := s.persist(&dup); err != nil {
		return Subgraph{}, err
	}
	return dup, nil
}

// GCAnonymous 匿名子图 mark-sweep: IsAnonymous && 不在 referenced 集合 → 删.
// referenced = 扫描时点全部容器图 + 全池子图图体引用到的子图 ID 集合 (由 wire 层提供,
// 那里有容器访问权). 触发点: 启动时 + 容器删除完成后 (锁序: Container → Subgraph).
// 具名子图永不自动删 (用户资产).
func (s *SubgraphStore) GCAnonymous(referenced map[string]bool) (removed []string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, sg := range s.byID {
		if !sg.IsAnonymous || referenced[id] {
			continue
		}
		if rmErr := s.removeLocked(id); rmErr != nil {
			return removed, rmErr
		}
		removed = append(removed, id)
	}
	return removed, nil
}
