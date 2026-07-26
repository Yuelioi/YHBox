package services

import (
	"github.com/yottaapp/yotta/internal/appcontrol"
)

// ApplicationService owns executable inspection. Configured applications are
// immediately available to workflows; this RPC never launches them.
type ApplicationService struct{ app *App }

func NewApplicationService(app *App) *ApplicationService { return &ApplicationService{app: app} }

func (s *ApplicationService) InspectExecutable(path string) (appcontrol.ExecutableInspection, error) {
	return appcontrol.InspectExecutable(path)
}
