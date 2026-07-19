package snippet

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	store *Store
	emit  func(name string, data any)
}

func NewService(store *Store, emit ...func(name string, data any)) *Service {
	service := &Service{store: store}
	if len(emit) > 0 {
		service.emit = emit[0]
	}
	return service
}

func (s *Service) List() ListResult {
	if s.store == nil {
		return ListResult{Items: []Summary{}, Warnings: []LoadWarning{{Error: "snippet store is unavailable"}}}
	}
	return s.store.List()
}

func (s *Service) Get(id string) (*Snippet, error) {
	if s.store == nil {
		return nil, errors.New("snippet store is unavailable")
	}
	value, ok := s.store.Get(strings.TrimSpace(id))
	if !ok {
		return nil, fmt.Errorf("snippet %q not found", id)
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
	value, ok := s.store.Get(strings.TrimSpace(id))
	if !ok {
		return nil, fmt.Errorf("snippet %q not found", id)
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

func (s *Service) emitChanged() {
	if s.emit != nil {
		s.emit("snippet:changed", map[string]any{})
	}
}
