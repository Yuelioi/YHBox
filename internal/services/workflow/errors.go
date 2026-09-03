package workflow

import (
	"fmt"

	"github.com/yottaapp/yotta/internal/apperr"
	"github.com/yottaapp/yotta/internal/serviceproblem"
)

func projectError(id, category string, params map[string]any, retryable bool, cause error) error {
	return serviceproblem.Wrap(id, category, params, retryable, cause)
}

func sourceError(operation string, cause error) error {
	return projectError("workflow.source.failed", apperr.CategoryInfrastructure, map[string]any{"operation": operation}, true, cause)
}

func bundleError(operation string, cause error) error {
	return projectError("workflow.bundle.failed", apperr.CategoryInfrastructure, map[string]any{"operation": operation}, true, cause)
}

func runError(operation string, cause error) error {
	return projectError("workflow.run.failed", apperr.CategoryDomain, map[string]any{"operation": operation}, false, cause)
}

func unavailable(feature string) error {
	return projectError("workflow.feature.unavailable", apperr.CategoryInfrastructure, map[string]any{"feature": feature}, true, fmt.Errorf("%s unavailable", feature))
}
