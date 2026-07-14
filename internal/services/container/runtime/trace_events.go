package runtime

import automationtrace "github.com/yottaapp/yotta/internal/automation/trace"

const actionTraceEventName = "container:action-trace"

type emittingTraceRecorder struct {
	base        automationtrace.Recorder
	containerID string
	emit        func(name string, data any)
	enabled     func() bool
}

func (r emittingTraceRecorder) Record(record automationtrace.ActionRecord) {
	if r.enabled != nil && !r.enabled() {
		return
	}
	if r.base != nil {
		r.base.Record(record)
	}
	if r.emit == nil {
		return
	}
	r.emit(actionTraceEventName, actionTracePayload(r.containerID, record))
}

func actionTracePayload(containerID string, record automationtrace.ActionRecord) map[string]any {
	duration := record.Duration()
	return map[string]any{
		"containerId":     containerID,
		"action":          record.Action,
		"source":          record.Source,
		"target":          record.Target,
		"backend":         record.Backend,
		"request":         record.Request,
		"result":          record.Result,
		"status":          record.Status,
		"error":           record.Error,
		"coordinateSteps": record.CoordinateSteps,
		"startedAt":       record.StartedAt,
		"endedAt":         record.EndedAt,
		"durationMs":      duration.Milliseconds(),
	}
}
