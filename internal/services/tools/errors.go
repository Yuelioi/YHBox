package tools

import "github.com/yottaapp/yotta/internal/serviceproblem"

func toolError(id, category string, params map[string]any, retryable bool, cause error) error {
	return serviceproblem.Wrap(id, category, params, retryable, cause)
}
