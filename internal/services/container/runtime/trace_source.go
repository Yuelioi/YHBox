package runtime

import automationtrace "github.com/yottaapp/yotta/internal/automation/trace"

type sourceTraceRecorder struct {
	base   automationtrace.Recorder
	source automationtrace.ActionSource
}

func (r sourceTraceRecorder) Record(record automationtrace.ActionRecord) {
	if r.base == nil {
		return
	}
	if record.Source.ContainerID == "" {
		record.Source.ContainerID = r.source.ContainerID
	}
	if record.Source.NodeID == "" {
		record.Source.NodeID = r.source.NodeID
	}
	if record.Source.NodeKind == "" {
		record.Source.NodeKind = r.source.NodeKind
	}
	if record.Source.InPin == "" {
		record.Source.InPin = r.source.InPin
	}
	r.base.Record(record)
}

func traceRecorderWithSource(base automationtrace.Recorder, source automationtrace.ActionSource) automationtrace.Recorder {
	if source.ContainerID == "" && source.NodeID == "" && source.NodeKind == "" && source.InPin == "" {
		return base
	}
	return sourceTraceRecorder{base: base, source: source}
}
