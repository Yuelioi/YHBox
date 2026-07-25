package schedule

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
)

// ChangeListener Schedule CRUD 后回调，main.go 注入 daemon.Reload。
type ChangeListener func() error
type TargetReadiness func(installationID string) error
type ServiceOption func(*Service)

func WithChangeListener(listener ChangeListener) ServiceOption {
	return func(service *Service) { service.onChange = listener }
}

func WithTargetReadiness(readiness TargetReadiness) ServiceOption {
	return func(service *Service) { service.readiness = readiness }
}

// PostCommitError means durable schedule state changed successfully, but the
// live daemon could not fully reload it. Callers must not blindly retry the
// persistence operation; surface the reload failure and reconcile live state.
type PostCommitError struct {
	Operation string
	Err       error
}

func (e *PostCommitError) Error() string {
	return fmt.Sprintf("schedule %s committed, but live reload failed: %v", e.Operation, e.Err)
}
func (e *PostCommitError) Unwrap() error   { return e.Err }
func (e *PostCommitError) Committed() bool { return true }

type Service struct {
	store     *Store
	onChange  ChangeListener
	readiness TargetReadiness
}

func NewService(store *Store, options ...ServiceOption) *Service {
	service := &Service{store: store}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	return service
}

func (s *Service) emitChange() error {
	if s.onChange != nil {
		return s.onChange()
	}
	return nil
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
// Schedule 创建时 targets 必须非空（validate 拦），
// Create 直接 Save 会 fail。所以两阶段：先 Create 拿 default，再 user 补 targets 后 Save。
func (s *Service) Create(name string) (Schedule, error) {
	sc := Schedule{
		SchemaVersion: CurrentSchemaVersion,
		ID:            uuid.NewString(),
		Name:          name,
		Enabled:       false,
		Targets:       []TargetRef{},
		Trigger:       Trigger{Kind: TriggerManual},
		OnError:       OnErrorStop,
	}
	return sc, nil
}

// Save 持久化一个完整 Schedule（首次或后续）。
func (s *Service) Save(sc Schedule) error {
	if err := s.requireEnabledTargetsReady(sc); err != nil {
		return err
	}
	if err := s.store.Save(&sc); err != nil {
		return err
	}
	if err := s.emitChange(); err != nil {
		return &PostCommitError{Operation: "save", Err: err}
	}
	return nil
}

// Update partial JSON 合并 + 保存。防 patch 改 ID。
func (s *Service) Update(id string, patchJSON string) error {
	sc, ok := s.store.Get(id)
	if !ok {
		return fmt.Errorf("schedule %q not found", id)
	}
	decoder := json.NewDecoder(strings.NewReader(patchJSON))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&sc); err != nil {
		return fmt.Errorf("parse patchJSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("parse patchJSON: expected exactly one JSON object")
	}
	sc.ID = id // 防 patch 改 ID 触发 path traversal
	if err := s.requireEnabledTargetsReady(sc); err != nil {
		return err
	}
	if err := s.store.Save(&sc); err != nil {
		return err
	}
	if err := s.emitChange(); err != nil {
		return &PostCommitError{Operation: "update", Err: err}
	}
	return nil
}

func (s *Service) requireEnabledTargetsReady(schedule Schedule) error {
	if !schedule.Enabled {
		return nil
	}
	if s.readiness == nil {
		return errors.New("Workflow Installation readiness is unavailable")
	}
	for _, target := range schedule.Targets {
		if target.Kind != TargetWorkflowInstallation {
			continue
		}
		if err := s.readiness(target.ID); err != nil {
			return fmt.Errorf("Workflow Installation %q is not ready for schedule: %w", target.ID, err)
		}
	}
	return nil
}

func (s *Service) Delete(id string) error {
	if err := s.store.Delete(id); err != nil {
		return err
	}
	if err := s.emitChange(); err != nil {
		return &PostCommitError{Operation: "delete", Err: err}
	}
	return nil
}
