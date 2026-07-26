package desktopapp

import (
	"context"
	"errors"

	"github.com/yottaapp/yotta/internal/application"
)

type workflowRunStarter struct{ application *application.Application }

func (s *workflowRunStarter) StartWorkflow(ctx context.Context, workflowID string) error {
	if s == nil || s.application == nil {
		return errors.New("Workflow runtime is unavailable")
	}
	_, err := s.application.StartRun(ctx, application.StartRunRequest{
		WorkflowID: workflowID,
		Principal:  "schedule",
	})
	return err
}
