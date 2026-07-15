package main

import (
	"context"
	"errors"

	app31 "github.com/yottaapp/yotta/internal/application"
)

type workflowRunStarter struct{ application *app31.Application }

func (s *workflowRunStarter) StartWorkflow(ctx context.Context, workflowID string) error {
	if s == nil || s.application == nil {
		return errors.New("workflow Application is unavailable")
	}
	_, err := s.application.StartRun(ctx, app31.StartRunRequest{WorkflowID: workflowID, Principal: "local-user"})
	return err
}
