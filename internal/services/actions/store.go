package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// v1Fields 已废的 Action 字段名（容器架构后砍除）。任一字段在 action.json 里
// 存在 → 视为 v1 数据，启动期整 root wipe。早期阶段无兼容性，用户已被告知重置。
var v1Fields = []string{
	"hotkey", "loopMode", "loopCount", "category", "tags", "description",
	"author", "source", "homepage", "version", "requiresForeground",
	"variables", "builtin",
}

// WipeV1Data 启动期调一次。扫 root 下所有 <id>/action.json，
// 任一含 v1 字段 → 删整个 root（含 per-action templates/）。返 wiped=true。
// 全部 v2-clean 或 root 不存在 → 返 wiped=false 不动数据。
//
// 简单粗暴：v1 检测命中一次就清光，不做 per-action 区分。
func WipeV1Data(root string) (wiped bool, err error) {
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		path := filepath.Join(root, ent.Name(), "action.json")
		b, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return false, err
		}
		var raw map[string]any
		if err := json.Unmarshal(b, &raw); err != nil {
			// parse 失败也算损坏，wipe 重来
			if rmErr := os.RemoveAll(root); rmErr != nil {
				return false, rmErr
			}
			return true, os.MkdirAll(root, 0o755)
		}
		for _, f := range v1Fields {
			if _, has := raw[f]; has {
				if rmErr := os.RemoveAll(root); rmErr != nil {
					return false, rmErr
				}
				return true, os.MkdirAll(root, 0o755)
			}
		}
	}
	return false, nil
}

// Store 把 actions 持久化到目录 —— **每个 action 一个文件夹**：
//
//	actions/<id>/
//	  action.json    # 元数据 + steps + variables
//	  README.md      # 长说明（可选）
//	  templates/     # 视觉决策模板图（Phase 1 开始用）
//	    _index.json  # 模板 metadata 索引
//	    <name>.png   # 模板 PNG
//
// 项目未发布，**不做** v1 单文件兼容。直接破坏式 schema。
type Store struct {
	mu      sync.RWMutex
	dir     string
	actions map[string]*Action
}

// NewStore dir 是存 action 文件夹的根目录。构造时 MkdirAll + load。
func NewStore(dir string) *Store {
	s := &Store{dir: dir, actions: make(map[string]*Action)}
	if err := os.MkdirAll(dir, 0755); err == nil {
		s.load()
	}
	return s
}

// load 扫 dir 下所有子文件夹的 action.json，解析成 Action 装进 memory。
// 单个文件夹损坏 → 跳过其它继续。
func (s *Store) load() {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		actionPath := filepath.Join(s.dir, e.Name(), "action.json")
		data, err := os.ReadFile(actionPath)
		if err != nil {
			continue
		}
		var a Action
		if err := json.Unmarshal(data, &a); err != nil {
			continue
		}
		if a.ID == "" {
			continue
		}
		s.actions[a.ID] = &a
	}
}

// ActionDir 返回某 action 的目录路径（actions/<id>/）。
func (s *Store) ActionDir(id string) string {
	return filepath.Join(s.dir, id)
}

// TemplatesDir 返回某 action 的 templates/ 目录路径。Phase 1 用。
func (s *Store) TemplatesDir(id string) string {
	return filepath.Join(s.dir, id, "templates")
}

// ReadmePath 返回某 action 的 README.md 路径。
func (s *Store) ReadmePath(id string) string {
	return filepath.Join(s.dir, id, "README.md")
}

// saveOneLocked 单个 action 落盘到 actions/<id>/action.json。caller 持 s.mu。
func (s *Store) saveOneLocked(a *Action) error {
	dir := s.ActionDir(a.ID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(dir, "action.json"), data)
}

// removeFolderLocked 删 actions/<id>/ 整个文件夹。caller 持 s.mu。
func (s *Store) removeFolderLocked(id string) error {
	err := os.RemoveAll(s.ActionDir(id))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func writeAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) List() []Action {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Action, 0, len(s.actions))
	for _, a := range s.actions {
		out = append(out, *a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

func (s *Store) Get(id string) (Action, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.actions[id]
	if !ok {
		return Action{}, false
	}
	return *a, true
}

func (s *Store) Create(a *Action) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a.ID == "" {
		return fmt.Errorf("ID 必须由 caller 分配（uuid）")
	}
	if _, dup := s.actions[a.ID]; dup {
		return fmt.Errorf("action id %q 已存在", a.ID)
	}
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	cp := *a
	s.actions[a.ID] = &cp
	return s.saveOneLocked(&cp)
}

func (s *Store) Update(id string, mutator func(*Action) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.actions[id]
	if !ok {
		return fmt.Errorf("action %q not found", id)
	}
	cp := *a
	if err := mutator(&cp); err != nil {
		return err
	}
	cp.UpdatedAt = time.Now()
	s.actions[id] = &cp
	return s.saveOneLocked(&cp)
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.actions[id]; !ok {
		return fmt.Errorf("action %q not found", id)
	}
	delete(s.actions, id)
	return s.removeFolderLocked(id)
}
