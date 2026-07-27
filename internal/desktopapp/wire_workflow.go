package desktopapp

import (
	"context"
	"errors"

	"github.com/yottaapp/yotta/internal/application"
	"github.com/yottaapp/yotta/internal/services/schedule"
)

type workflowRunStarter struct{ application *application.Application }

func (s *workflowRunStarter) StartWorkflow(ctx context.Context, workflowID string) (schedule.RunReadiness, error) {
	if s == nil || s.application == nil {
		return schedule.RunReadiness{State: string(application.RunReadinessFailed)}, errors.New("Workflow runtime is unavailable")
	}
	result, err := s.application.StartRun(ctx, application.StartRunRequest{
		WorkflowID: workflowID,
		Principal:  "schedule",
	})
	readiness := application.ClassifyRunStart(result, err)
	view := schedule.RunReadiness{
		State: string(readiness.State), Code: readiness.Code, GraphID: readiness.GraphID,
		NodeID: readiness.NodeID, RequirementID: readiness.RequirementID, Slot: readiness.Slot,
	}
	if readiness.UserFixable() {
		return view, nil
	}
	return view, err
}
