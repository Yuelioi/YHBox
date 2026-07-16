package main

import (
	"context"
	"errors"

	appcore "github.com/yottaapp/yotta/internal/application"
)

type workflowRunStarter struct{ application *appcore.Application }

func (s *workflowRunStarter) StartWorkflow(ctx context.Context, workflowID string) error {
	if s == nil || s.application == nil {
		return errors.New("workflow Application is unavailable")
	}
	_, err := s.application.StartRun(ctx, appcore.StartRunRequest{WorkflowID: workflowID, Principal: "local-user"})
	return err
}
