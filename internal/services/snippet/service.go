package snippet

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yottaapp/yotta/internal/apperr"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	workflowauthoring "github.com/yottaapp/yotta/internal/workflow/authoring"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

type Service struct {
	store     *Store
	authoring nodeauthoring.Snapshot
	emit      func(name string, data any)
}

func NewService(store *Store, emit ...func(name string, data any)) *Service {
	service := &Service{store: store}
	if len(emit) > 0 {
		service.emit = emit[0]
	}
	return service
}

// NewServiceWithAuthoring enables durable NodeRef migration for persisted
// snippets. The migration runs only when an old snippet is read or saved; it
// is not part of workflow execution.
func NewServiceWithAuthoring(store *Store, authoring nodeauthoring.Snapshot, emit ...func(name string, data any)) *Service {
	service := NewService(store, emit...)
	service.authoring = authoring
	return service
}

func (s *Service) List() ListResult {
	if s.store == nil {
		envelope := apperr.From(apperr.New("snippet.store.unavailable", nil))
		return ListResult{Items: []Summary{}, Warnings: []LoadWarning{{Problem: &envelope}}}
	}
	return s.store.List()
}

func (s *Service) Get(id string) (*Snippet, error) {
	if s.store == nil {
		return nil, errors.New("snippet store is unavailable")
	}
	value, err := s.load(strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func (s *Service) Save(value *Snippet) (*Snippet, error) {
	if s.store == nil || value == nil {
		return nil, errors.New("snippet service requires a store and value")
	}
	result := clone(*value)
	result.SchemaVersion = SchemaVersion
	result.Name = strings.TrimSpace(result.Name)
	result.Description = strings.TrimSpace(result.Description)
	result.Category = strings.TrimSpace(result.Category)
	result.Tags = normalizeTags(result.Tags)
	shortcut, err := normalizeShortcut(result.Shortcut)
	if err != nil {
		return nil, err
	}
	result.Shortcut = shortcut
	if _, err := s.upgradeNodeTemplate(&result); err != nil {
		return nil, err
	}
	if shortcut != "" {
		for _, item := range s.store.List().Items {
			if item.ID != result.ID && strings.EqualFold(item.Shortcut, shortcut) {
				return nil, fmt.Errorf("snippet shortcut %q is already used by %q", shortcut, item.Name)
			}
		}
	}
	now := time.Now().UTC()
	if strings.TrimSpace(result.ID) == "" {
		result.ID = "snippet-" + uuid.NewString()
		result.CreatedAt = now
	} else if existing, ok := s.store.Get(result.ID); ok {
		result.CreatedAt = existing.CreatedAt
	} else if result.CreatedAt.IsZero() {
		result.CreatedAt = now
	}
	result.UpdatedAt = now
	if err := s.store.Save(result); err != nil {
		return nil, fmt.Errorf("save snippet: %w", err)
	}
	s.emitChanged()
	saved := clone(result)
	return &saved, nil
}

func (s *Service) Delete(id string) error {
	if s.store == nil {
		return errors.New("snippet store is unavailable")
	}
	if err := s.store.Delete(strings.TrimSpace(id)); err != nil {
		return err
	}
	s.emitChanged()
	return nil
}

func (s *Service) MarkUsed(id string) (*Snippet, error) {
	if s.store == nil {
		return nil, errors.New("snippet store is unavailable")
	}
	value, err := s.load(strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	value.UsageCount++
	value.LastUsedAt = &now
	if err := s.store.Save(value); err != nil {
		return nil, fmt.Errorf("mark snippet used: %w", err)
	}
	s.emitChanged()
	result := clone(value)
	return &result, nil
}

func (s *Service) load(id string) (Snippet, error) {
	value, ok := s.store.Get(id)
	if !ok {
		return Snippet{}, fmt.Errorf("snippet %q not found", id)
	}
	changed, err := s.upgradeNodeTemplate(&value)
	if err != nil {
		return Snippet{}, fmt.Errorf("snippet %q: %w", id, err)
	}
	if changed {
		if err := s.store.Save(value); err != nil {
			return Snippet{}, fmt.Errorf("persist migrated snippet %q: %w", id, err)
		}
	}
	return value, nil
}

func (s *Service) upgradeNodeTemplate(value *Snippet) (bool, error) {
	if !s.authoring.Valid() {
		return false, nil
	}
	upgraded, changed, err := workflowauthoring.UpgradeDetachedNode(schema.Node{
		NodeRef:  value.Payload.NodeRef,
		Label:    value.Payload.Label,
		Config:   value.Payload.Config,
		Bindings: value.Payload.Bindings,
		Disabled: value.Payload.Disabled,
	}, s.authoring)
	if err != nil {
		return false, fmt.Errorf("node %q is incompatible with this Yotta version: %w", value.Payload.NodeRef.NodeTypeID, err)
	}
	if !changed {
		return false, nil
	}
	value.Payload.NodeRef = upgraded.NodeRef
	value.Payload.Label = upgraded.Label
	value.Payload.Config = upgraded.Config
	value.Payload.Bindings = upgraded.Bindings
	value.Payload.Disabled = upgraded.Disabled
	return true, nil
}

func (s *Service) emitChanged() {
	if s.emit != nil {
		s.emit("snippet:changed", map[string]any{})
	}
}
