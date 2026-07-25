package desktopapp

import (
	"context"
	"errors"

	"github.com/yottaapp/yotta/internal/appbootstrap"
	"github.com/yottaapp/yotta/internal/workflowinstallation"
)

type workflowRunStarter struct{ runtime *appbootstrap.Runtime }

func (s *workflowRunStarter) StartInstallation(ctx context.Context, installationID string) error {
	if s == nil || s.runtime == nil {
		return errors.New("Workflow Installation runtime is unavailable")
	}
	_, err := s.runtime.StartInstallationRun(ctx, installationID, workflowinstallation.ScopeSchedule)
	return err
}
