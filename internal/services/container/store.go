package container

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

var idRE = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// validateID 检查 ID 是否安全做 folder 名（防 path traversal）。
// 接受 alphanumeric + 下划线 + 减号；拒绝空、含 / 含 \ 含 .. 含 : 等。
func validateID(id string) error {
	if id == "" {
		return errors.New("container.id 不能为空")
	}
	if !idRE.MatchString(id) {
		return fmt.Errorf("container.id %q 含非法字符（只允许字母/数字/_/-）", id)
	}
	return nil
}

// Store 文件系统存储：data/containers/<id>/container.json。
type Store struct {
	mu   sync.RWMutex
	root string
	byID map[string]Container
	// resolveSubgraphs 容器 → 引用闭包子图集 (全局池来源, wire 层注入; 校验用).
	// nil → 按空闭包校验 (纯测试/工具语境). 调用时持本 store 锁 → 全局锁序 Container → Subgraph.
	resolveSubgraphs func(c *Container) []Subgraph
}

// SetSubgraphResolver 注入容器引用闭包解析 (main.go wire; 见 Store.resolveSubgraphs).
func (s *Store) SetSubgraphResolver(f func(c *Container) []Subgraph) {
	s.resolveSubgraphs = f
}

// subgraphsFor 解析容器引用闭包; resolver 未注入 → nil.
func (s *Store) subgraphsFor(c *Container) []Subgraph {
	if s.resolveSubgraphs == nil {
		return nil
	}
	return s.resolveSubgraphs(c)
}

func NewStore(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", root, err)
	}
	s := &Store{root: root, byID: map[string]Container{}}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return fmt.Errorf("readdir: %w", err)
	}
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		c, err := s.loadOne(ent.Name())
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		s.byID[c.ID] = c
	}
	return nil
}

// loadOne 读单个容器目录 (<root>/<id>/container.json; 子图已全局化, 不在容器目录)。
// 不加锁 —— caller (load 构造期 / Reload 持写锁) 负责并发安全。
//   - 目录或 container.json 不存在 → 返 os.ErrNotExist (caller 区分: load skip / Reload 删 byID)。
//   - 读失败 / JSON 解析失败 → 返 StatusIncompatible 占位 Container + nil error (不阻断, 与原 load 容错一致)。
func (s *Store) loadOne(id string) (Container, error) {
	path := filepath.Join(s.root, id, "container.json")
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Container{}, err
	}
	if err != nil {
		return Container{
			ID:                 id,
			Name:               id,
			Status:             StatusIncompatible,
			IncompatibleReason: fmt.Sprintf("读取失败：%v", err),
		}, nil
	}
	var c Container
	if err := json.Unmarshal(b, &c); err != nil {
		return Container{
			ID:                 id,
			Name:               id,
			Status:             StatusIncompatible,
			IncompatibleReason: fmt.Sprintf("JSON 解析失败：%v", err),
		}, nil
	}
	// Graph.SchemaVersion 检查
	// > GraphSchemaVersion → 未来版本（真 incompatible）
	// == 0 → 旧数据（v1 / Create 早期 bug 漏写），auto-upgrade 不阻塞使用
	switch {
	case c.Graph.SchemaVersion > GraphSchemaVersion:
		c.Status = StatusIncompatible
		c.IncompatibleReason = fmt.Sprintf("graph version=%d 不支持（当前 %d）", c.Graph.SchemaVersion, GraphSchemaVersion)
	case c.Graph.SchemaVersion == 0:
		c.Graph.SchemaVersion = GraphSchemaVersion
		if c.Graph.ID == "" {
			c.Graph.ID = "g-" + id
		}
	}
	// 子图已全局化 (data/subgraphs/) — 容器目录不再有 subgraphs/, 这里不加载.
	return c, nil
}

// Reload 从磁盘重读单个容器 (container.json), 替换内存缓存。
// 配合 MCP / 外部进程改盘后, 编辑器「重载」按钮调用 (走 Service.Reload RPC)。
// 容器目录已不存在 → 从 byID 删除并返 not-found error。
func (s *Store) Reload(id string) (Container, error) {
	if err := validateID(id); err != nil {
		return Container{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c, err := s.loadOne(id)
	if errors.Is(err, os.ErrNotExist) {
		delete(s.byID, id)
		return Container{}, fmt.Errorf("container %q 不存在（可能已被删除）", id)
	}
	if err != nil {
		return Container{}, err
	}
	s.byID[c.ID] = c
	return c, nil
}

// Save 持久化（含 validate）。
func (s *Store) Save(c *Container) error {
	if err := validateID(c.ID); err != nil {
		return err
	}
	// 本地副本：避免 mutation 通过指针泄漏到 caller，避免共享指针 race
	local := *c
	local.Normalize()
	// 注: 此时未持本 store 锁, resolver 自由拿子图 store 读锁 (锁序无忧).
	if err := local.Validate(s.subgraphsFor(&local)); err != nil {
		return err
	}
	now := time.Now().UTC()
	if local.CreatedAt.IsZero() {
		local.CreatedAt = now
	}
	local.UpdatedAt = now

	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Join(s.root, local.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(local, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, "container.json.tmp")
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, filepath.Join(dir, "container.json")); err != nil {
		_ = os.Remove(tmp) // 清理孤立 tmp
		return err
	}
	s.byID[local.ID] = local
	return nil
}

func (s *Store) Get(id string) (Container, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.byID[id]
	return c, ok
}

func (s *Store) List() []Container {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Container, 0, len(s.byID))
	for _, c := range s.byID {
		out = append(out, c)
	}
	return out
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := filepath.Join(s.root, id)
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	delete(s.byID, id)
	return nil
}
