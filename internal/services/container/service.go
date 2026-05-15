package container

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// Runner Container 运行入口。main.go 注入：把 RunOnce 转成"enqueue 单 target run"。
// StopAll 取消当前 worker run + 清队列。
type Runner interface {
	RunOnce(id string) error
	StopAll() error
}

// ChangeListener 容器 CRUD 后回调，给 hotkey registry / schedule daemon 重新注册用。
type ChangeListener func()

// Service wails3 RPC 入口。
type Service struct {
	store    *Store
	runner   Runner
	onChange ChangeListener
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

// SetRunner 启动期 main.go 注入。Runner=nil 时 Run/Stop 返 error。
func (s *Service) SetRunner(r Runner) { s.runner = r }

// SetOnChange 启动期 main.go 注入。CRUD 后调一次（保存成功才调）。
func (s *Service) SetOnChange(f ChangeListener) { s.onChange = f }

func (s *Service) emitChange() {
	if s.onChange != nil {
		s.onChange()
	}
}

func (s *Service) List() []Container {
	return s.store.List()
}

func (s *Service) Get(id string) (Container, error) {
	c, ok := s.store.Get(id)
	if !ok {
		return Container{}, fmt.Errorf("container %q not found", id)
	}
	return c, nil
}

// Create 新建一个空 Container：自带 Start → Stop 骨架，用户直接在中间插节点。
// 这种"前置骨架"对新手更友好（看到容器有明确起止），但 Stop 不是必须 — 删了也能跑。
func (s *Service) Create(name string) (Container, error) {
	startID := uuid.NewString()
	stopID := uuid.NewString()
	c := Container{
		SchemaVersion: CurrentSchemaVersion,
		ID:            uuid.NewString(),
		Name:          name,
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: startID, Kind: "Start", X: 100, Y: 120, Config: map[string]any{}},
				{ID: stopID, Kind: "Stop", X: 500, Y: 120, Config: map[string]any{}},
			},
			Edges: []GraphEdge{
				{From: startID + ".out", To: stopID + ".in"},
			},
		},
	}
	if err := s.store.Save(&c); err != nil {
		return Container{}, err
	}
	saved, _ := s.store.Get(c.ID)
	s.emitChange()
	return saved, nil
}

// Update patchJSON partial JSON merge 到现 Container。
func (s *Service) Update(id string, patchJSON string) error {
	c, ok := s.store.Get(id)
	if !ok {
		return fmt.Errorf("container %q not found", id)
	}
	if err := json.Unmarshal([]byte(patchJSON), &c); err != nil {
		return fmt.Errorf("parse patchJSON: %w", err)
	}
	c.ID = id // 防 patch 改 ID 触发 path traversal
	if err := s.store.Save(&c); err != nil {
		return err
	}
	s.emitChange()
	return nil
}

func (s *Service) Delete(id string) error {
	if err := s.store.Delete(id); err != nil {
		return err
	}
	s.emitChange()
	return nil
}

// Run 立即跑一次（前端 ▶ 按钮）。manual source 入 ExecutionQueue 单 target run。
func (s *Service) Run(id string) error {
	if _, ok := s.store.Get(id); !ok {
		return fmt.Errorf("container %q not found", id)
	}
	if s.runner == nil {
		return fmt.Errorf("runner not injected")
	}
	return s.runner.RunOnce(id)
}

// StopAll UI 全局停按钮（跟 Ctrl+Shift+F9 同语义）。
func (s *Service) StopAll() error {
	if s.runner == nil {
		return fmt.Errorf("runner not injected")
	}
	return s.runner.StopAll()
}
