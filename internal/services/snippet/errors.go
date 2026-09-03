package snippet

import "github.com/yottaapp/yotta/internal/serviceproblem"

func problem(id, category string, params map[string]any, retryable bool, cause error) error {
	return serviceproblem.Wrap(id, category, params, retryable, cause)
}
