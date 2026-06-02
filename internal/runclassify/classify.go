// Package runclassify 把 run error 分类成 execution.RunError 事件信封。
// 单独成包: import execution + container + apperr 而不造成 execution→container 循环。
package runclassify

import (
	"errors"

	"yotta/internal/apperr"
	"yotta/internal/services/container"
	"yotta/internal/services/execution"
)

// RunError 把 run 失败 error 分类成结构化信封。nil error → nil。
func RunError(err error) *execution.RunError {
	if err == nil {
		return nil
	}
	var vf *container.ValidationFailure
	if errors.As(err, &vf) {
		re := &execution.RunError{}
		for _, e := range vf.Errors {
			re.Errors = append(re.Errors, execution.RunValErr{
				Severity: e.Severity, Code: e.Code, GraphPath: e.GraphPath,
				NodeID: e.NodeID, Params: e.Params,
			})
		}
		return re
	}
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return &execution.RunError{Code: ae.Code, Params: ae.Params}
	}
	return &execution.RunError{Message: err.Error()}
}
