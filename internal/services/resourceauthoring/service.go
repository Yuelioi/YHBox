package resourceauthoring

import (
	"context"

	"github.com/yottaapp/yotta/internal/workflow/schema"
)

// Service is the Wails adapter for editor-owned image creation.
type Service struct {
	creator *Creator
	emit    func(name string, data any)
}

func NewService(creator *Creator, emit ...func(name string, data any)) *Service {
	service := &Service{creator: creator}
	if len(emit) != 0 {
		service.emit = emit[0]
	}
	return service
}

func (s *Service) CreateImage(draft ImageDraft) (schema.WorkflowResource, error) {
	return s.creator.CreateImage(context.Background(), draft)
}

func (s *Service) Open(resource schema.WorkflowResource) (Content, error) {
	return s.creator.Open(context.Background(), resource)
}

func (s *Service) Rewrite(resource schema.WorkflowResource, edit Edit) (schema.WorkflowResource, error) {
	return s.creator.Rewrite(context.Background(), resource, edit)
}

func (s *Service) Events(resource schema.WorkflowResource, offset, limit int) (EventPage, error) {
	return s.creator.Events(context.Background(), resource, offset, limit)
}

func (s *Service) Duplicate(resource schema.WorkflowResource) (schema.WorkflowResource, error) {
	return s.creator.Duplicate(context.Background(), resource)
}

func (s *Service) Promote(resource schema.WorkflowResource) (Promotion, error) {
	promotion, err := s.creator.Promote(context.Background(), resource)
	if err == nil && s.emit != nil {
		s.emit("asset:changed", map[string]any{
			"revision": promotion.Revision,
			"guids":    []string{promotion.GUID},
		})
	}
	return promotion, err
}
