package schedule

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// ChangeListener Schedule CRUD 后回调，main.go 注入 daemon.Reload。
type ChangeListener func()

type Service struct {
	store    *Store
	onChange ChangeListener
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

// SetOnChange 启动期 main.go 注入。Save/Update/Delete 成功后调一次。
func (s *Service) SetOnChange(f ChangeListener) { s.onChange = f }

func (s *Service) emitChange() {
	if s.onChange != nil {
		s.onChange()
	}
}

func (s *Service) List() []Schedule {
	return s.store.List()
}

func (s *Service) Get(id string) (Schedule, error) {
	sc, ok := s.store.Get(id)
	if !ok {
		return Schedule{}, fmt.Errorf("schedule %q not found", id)
	}
	return sc, nil
}

// Create 返一个 default Schedule，不持久化。前端拿到 default 让用户改 + 按"保存"调 Save。
//
// 区别于 container.Service.Create：Schedule 创建时 targets 必须非空（validate 拦），
// Create 直接 Save 会 fail。所以两阶段：先 Create 拿 default，再 user 补 targets 后 Save。
func (s *Service) Create(name string) (Schedule, error) {
	sc := Schedule{
		SchemaVersion: CurrentSchemaVersion,
		ID:            uuid.NewString(),
		Name:          name,
		Enabled:       true,
		Targets:       []TargetRef{},
		Trigger:       Trigger{Kind: "manual"},
		OnError:       "stop",
	}
	return sc, nil
}

// Save 持久化一个完整 Schedule（首次或后续）。
func (s *Service) Save(sc Schedule) error {
	if err := s.store.Save(&sc); err != nil {
		return err
	}
	s.emitChange()
	return nil
}

// Update partial JSON 合并 + 保存。防 patch 改 ID。
func (s *Service) Update(id string, patchJSON string) error {
	sc, ok := s.store.Get(id)
	if !ok {
		return fmt.Errorf("schedule %q not found", id)
	}
	if err := json.Unmarshal([]byte(patchJSON), &sc); err != nil {
		return fmt.Errorf("parse patchJSON: %w", err)
	}
	sc.ID = id // 防 patch 改 ID 触发 path traversal
	if err := s.store.Save(&sc); err != nil {
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
