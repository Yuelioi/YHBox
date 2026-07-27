package services

import (
	"github.com/yottaapp/yotta/internal/appcontrol"
)

// ApplicationService exposes executable inspection. Configured applications
// are immediately available to workflows; this RPC never launches them.
type ApplicationService struct{}

func NewApplicationService() *ApplicationService { return &ApplicationService{} }

func (s *ApplicationService) InspectExecutable(path string) (appcontrol.ExecutableInspection, error) {
	return appcontrol.InspectExecutable(path)
}
