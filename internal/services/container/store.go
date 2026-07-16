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

	nodepkg "github.com/yottaapp/yotta/internal/node"
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

// Store 文件系统存储：data/containers/<id>/{package.json,graph.json,installation.json,yotta-lock.json}。
type Store struct {
	mu   sync.RWMutex
	root string
	byID map[string]Container
	// resolveSubgraphs 容器 → 引用闭包子图集 (全局池来源, wire 层注入; 校验用).
	// nil → 按空闭包校验 (纯测试/工具语境). 调用时持本 store 锁 → 全局锁序 Container → Subgraph.
	resolveSubgraphs func(c *Container) []Subgraph
	registry         nodepkg.RegistrySnapshot
	writeFileAtomic  func(path string, data []byte) error
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
	return NewStoreWithRegistry(root, nodepkg.DefaultRegistrySnapshot())
}

func NewStoreWithRegistry(root string, registry nodepkg.RegistryReader) (*Store, error) {
	if registry == nil {
		return nil, errors.New("container: node registry is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", root, err)
	}
	s := &Store{root: root, byID: map[string]Container{}, registry: nodepkg.SnapshotRegistry(registry), writeFileAtomic: writeContainerFileAtomic}
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

// loadOne 读单个容器目录 (<root>/<id>/package.json + graph.json + installation.json; 子图已全局化, 不在容器目录)。
// 不加锁 —— caller (load 构造期 / Reload 持写锁) 负责并发安全。
//   - 目录或 package.json 不存在 → 返 os.ErrNotExist (caller 区分: load skip / Reload 删 byID)。
//   - 读失败 / JSON 解析失败 → 返 StatusIncompatible 占位 Container + nil error (不阻断, 与原 load 容错一致)。
func (s *Store) loadOne(id string) (Container, error) {
	dir := filepath.Join(s.root, id)
	manifest, err := readJSONFile[PackageManifest](filepath.Join(dir, packageFile))
	if errors.Is(err, os.ErrNotExist) {
		return Container{}, err
	}
	if err != nil {
		return Container{
			ID:                 id,
			Name:               id,
			Status:             StatusIncompatible,
			IncompatibleReason: fmt.Sprintf("读取 package.json 失败：%v", err),
		}, nil
	}
	graph, err := readJSONFile[Graph](filepath.Join(dir, graphFile))
	if err != nil {
		return Container{
			ID:                 id,
			Name:               manifest.DisplayName,
			Status:             StatusIncompatible,
			IncompatibleReason: fmt.Sprintf("读取 graph.json 失败：%v", err),
		}, nil
	}
	installation, err := readJSONFile[Installation](filepath.Join(dir, installationFile))
	if err != nil {
		return Container{
			ID:                 id,
			Name:               manifest.DisplayName,
			Status:             StatusIncompatible,
			IncompatibleReason: fmt.Sprintf("读取 installation.json 失败：%v", err),
		}, nil
	}
	lock, err := readJSONFile[YottaLock](filepath.Join(dir, lockFile))
	if err != nil {
		return incompatibleContainer(id, manifest.DisplayName, fmt.Sprintf("读取 yotta-lock.json 失败：%v", err)), nil
	}
	if err := validateContainerCommit(id, manifest, graph, installation, lock); err != nil {
		return incompatibleContainer(id, manifest.DisplayName, err.Error()), nil
	}
	if lock.SchemaVersion == 1 {
		lock.SchemaVersion = LockSchemaVersion
		lock.InstallationHash, err = hashJSON(installation)
		if err != nil {
			return incompatibleContainer(id, manifest.DisplayName, err.Error()), nil
		}
		data, err := json.MarshalIndent(lock, "", "  ")
		if err != nil {
			return incompatibleContainer(id, manifest.DisplayName, err.Error()), nil
		}
		// A valid v1 lock remains readable when the directory is read-only or an
		// opportunistic migration is interrupted. The next successful Save writes
		// v2 again, so migration failure must not make legacy data unusable.
		_ = s.writeFileAtomic(filepath.Join(dir, lockFile), data)
	}
	c := aggregateContainer(manifest, graph, installation)
	// Graph.SchemaVersion 检查
	// > GraphSchemaVersion → 未来版本（真 incompatible）
	// == 0 → 开发期临时数据漏写，auto-upgrade 不阻塞使用
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

// Reload 从磁盘重读单个容器, 替换内存缓存。
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
	clone, err := cloneContainer(c)
	return clone, err
}

// Save 持久化（含 validate）。
func (s *Store) Save(c *Container) error {
	if err := validateID(c.ID); err != nil {
		return err
	}
	// 本地副本：避免 mutation 通过指针泄漏到 caller，避免共享指针 race
	local, err := cloneContainer(*c)
	if err != nil {
		return fmt.Errorf("clone container: %w", err)
	}
	local.Normalize()
	// 注: 此时未持本 store 锁, resolver 自由拿子图 store 读锁 (锁序无忧).
	if err := local.ValidateWithRegistry(s.subgraphsFor(&local), s.registry); err != nil {
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
	portableGraph, targetBindings, aiBindings := splitPortableGraphBindings(local.Graph)
	manifest := containerToPackageManifest(local)
	manifest.Yotta.Targets = targetSlotsFromGraph(portableGraph)
	manifest.Yotta.AI = aiSlotsFromGraph(portableGraph)
	installation := containerToInstallation(local, manifest)
	installation.TargetBindings = targetBindings
	installation.AIBindings = aiBindings
	lock, err := buildContainerLockWithRegistry(s.registry, manifest, portableGraph, s.subgraphsFor(&local), now.Format(time.RFC3339))
	if err != nil {
		return err
	}
	lock.InstallationHash, err = hashJSON(installation)
	if err != nil {
		return err
	}
	files := []containerCommitFile{
		{name: packageFile, value: manifest},
		{name: graphFile, value: portableGraph},
		{name: installationFile, value: installation},
		{name: lockFile, value: lock},
	}
	if err := s.commitFiles(dir, files); err != nil {
		if containerRollbackFailed(err) {
			s.byID[local.ID] = incompatibleContainer(local.ID, local.Name,
				"容器保存回滚失败，磁盘状态不确定；请重新加载或恢复备份")
		}
		return err
	}
	_ = os.Remove(filepath.Join(dir, "container.json"))
	s.byID[local.ID] = aggregateContainer(manifest, portableGraph, installation)
	return nil
}

func incompatibleContainer(id, name, reason string) Container {
	if name == "" {
		name = id
	}
	return Container{ID: id, Name: name, Status: StatusIncompatible, IncompatibleReason: reason}
}

func (s *Store) Get(id string) (Container, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.byID[id]
	if !ok {
		return Container{}, false
	}
	clone, err := cloneContainer(c)
	return clone, err == nil
}

func (s *Store) List() []Container {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Container, 0, len(s.byID))
	for _, c := range s.byID {
		clone, err := cloneContainer(c)
		if err == nil {
			out = append(out, clone)
		}
	}
	return out
}

// RegistrySnapshot returns the node contract generation used by this store.
func (s *Store) RegistrySnapshot() nodepkg.RegistrySnapshot {
	return s.registry
}

func cloneContainer(c Container) (Container, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return Container{}, err
	}
	var clone Container
	if err := json.Unmarshal(data, &clone); err != nil {
		return Container{}, err
	}
	clone.Status = c.Status
	clone.IncompatibleReason = c.IncompatibleReason
	return clone, nil
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
