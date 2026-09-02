package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/yottaapp/yotta/internal/application"
	"github.com/yottaapp/yotta/internal/artifact"
	rundomain "github.com/yottaapp/yotta/internal/run"
)

const (
	RunListFormat      = "yotta.run-list"
	RunEvidenceFormat  = "yotta.run-evidence"
	RunEvidenceVersion = "1"
)

type RunListRequest struct {
	WorkflowID string `json:"workflowId,omitempty"`
	Status     string `json:"status,omitempty"`
	Offset     int    `json:"offset,omitempty" jsonschema:"minimum=0"`
	Limit      int    `json:"limit,omitempty" jsonschema:"minimum=1,maximum=100"`
}

type RunGetRequest struct {
	RunID    string `json:"runId" jsonschema:"required"`
	Page     int    `json:"page,omitempty" jsonschema:"minimum=1"`
	PageSize int    `json:"pageSize,omitempty" jsonschema:"minimum=1,maximum=500"`
}

type RunEvidenceSummary struct {
	RunID          string          `json:"runId"`
	WorkflowID     string          `json:"workflowId,omitempty"`
	SourceHash     artifact.Digest `json:"sourceHash,omitempty"`
	SourceRevision int64           `json:"sourceRevision,omitempty"`
	ProgramHash    artifact.Digest `json:"programHash"`
	Status         string          `json:"status"`
	QueuedAt       string          `json:"queuedAt"`
}

type RunListResult struct {
	Format  string               `json:"format"`
	Version string               `json:"version"`
	Offset  int                  `json:"offset"`
	Total   int                  `json:"total"`
	Items   []RunEvidenceSummary `json:"items"`
}

type RunProblem struct {
	ID        string         `json:"id"`
	Category  string         `json:"category"`
	Params    map[string]any `json:"params,omitempty"`
	Retryable bool           `json:"retryable"`
	GraphID   string         `json:"graphId,omitempty"`
	NodeID    string         `json:"nodeId,omitempty"`
	Attempt   int            `json:"attempt,omitempty"`
}

type RunEvidenceEntry struct {
	Sequence       uint64            `json:"sequence"`
	Kind           string            `json:"kind"`
	GraphPath      []string          `json:"graphPath"`
	NodeID         string            `json:"nodeId"`
	Attempt        int               `json:"attempt"`
	AttemptOutcome string            `json:"attemptOutcome,omitempty"`
	Action         string            `json:"action,omitempty"`
	ActionOutcome  string            `json:"actionOutcome,omitempty"`
	StatusID       string            `json:"statusId,omitempty"`
	StatusCategory string            `json:"statusCategory,omitempty"`
	ProblemID      string            `json:"problemId,omitempty"`
	ProblemParams  map[string]any    `json:"problemParams,omitempty"`
	SummaryID      string            `json:"summaryId"`
	Counters       map[string]int64  `json:"counters"`
	Facts          map[string]string `json:"facts,omitempty"`
	OccurredAt     string            `json:"occurredAt"`
}

type RunEvidenceResult struct {
	Format        string             `json:"format"`
	Version       string             `json:"version"`
	Run           RunEvidenceSummary `json:"run"`
	Problem       *RunProblem        `json:"problem,omitempty"`
	Timeline      []RunEvidenceEntry `json:"timeline"`
	TimelinePage  int                `json:"timelinePage"`
	TimelinePages int                `json:"timelinePages"`
	TimelineTotal int                `json:"timelineTotal"`
}

func listRuns(app *application.Application, request RunListRequest) (RunListResult, error) {
	offset, limit, err := boundedPage(request.Offset, request.Limit)
	if err != nil {
		return RunListResult{}, err
	}
	records, err := app.ListRuns()
	if err != nil {
		return RunListResult{}, err
	}
	items := make([]RunEvidenceSummary, 0, len(records))
	for _, record := range records {
		admission := record.Admission()
		if request.WorkflowID != "" && admission.WorkflowID != request.WorkflowID ||
			request.Status != "" && !strings.EqualFold(string(record.Status()), request.Status) {
			continue
		}
		items = append(items, runEvidenceSummary(record))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].QueuedAt > items[j].QueuedAt })
	return RunListResult{Format: RunListFormat, Version: RunEvidenceVersion, Offset: offset, Total: len(items), Items: page(items, offset, limit)}, nil
}

func getRunEvidence(ctx context.Context, app *application.Application, request RunGetRequest) (RunEvidenceResult, error) {
	if strings.TrimSpace(request.RunID) == "" {
		return RunEvidenceResult{}, errors.New("run.get.invalid_request")
	}
	pageNumber, pageSize := request.Page, request.PageSize
	if pageNumber == 0 {
		pageNumber = 1
	}
	if pageSize == 0 {
		pageSize = 200
	}
	timeline, err := app.GetRunTimelinePage(ctx, request.RunID, pageNumber, pageSize)
	if err != nil {
		return RunEvidenceResult{}, err
	}
	record, err := app.GetRun(request.RunID)
	if err != nil {
		return RunEvidenceResult{}, err
	}
	result := RunEvidenceResult{
		Format: RunEvidenceFormat, Version: RunEvidenceVersion, Run: runEvidenceSummary(record),
		TimelinePage: timeline.Page, TimelinePages: timeline.Pages, TimelineTotal: timeline.Total,
		Timeline: make([]RunEvidenceEntry, 0, len(timeline.Entries)),
	}
	if failure, ok := record.Failure(); ok {
		var params map[string]any
		if len(failure.Params) != 0 {
			_ = json.Unmarshal(failure.Params, &params)
		}
		result.Problem = &RunProblem{ID: failure.Code, Category: failure.Category, Params: params, Retryable: failure.Retryable, GraphID: failure.GraphID, NodeID: failure.NodeID, Attempt: failure.Attempt}
	}
	for _, entry := range timeline.Entries {
		var params map[string]any
		if len(entry.ErrorParams) != 0 {
			_ = json.Unmarshal(entry.ErrorParams, &params)
		}
		result.Timeline = append(result.Timeline, RunEvidenceEntry{
			Sequence: entry.Sequence, Kind: string(entry.Kind), GraphPath: append([]string(nil), entry.GraphPath...), NodeID: entry.NodeID,
			Attempt: entry.Attempt, AttemptOutcome: string(entry.AttemptOutcome), Action: entry.Action, ActionOutcome: string(entry.ActionOutcome),
			StatusID: entry.StatusCode, StatusCategory: string(entry.StatusCategory), ProblemID: entry.ErrorCode, ProblemParams: params,
			SummaryID: entry.Summary.Code, Counters: entry.Summary.Counters, Facts: entry.Summary.Facts,
			OccurredAt: entry.OccurredAt.Format("2006-01-02T15:04:05.999999999Z07:00"),
		})
	}
	return result, nil
}

func runEvidenceSummary(record rundomain.Record) RunEvidenceSummary {
	admission := record.Admission()
	return RunEvidenceSummary{RunID: admission.RunID, WorkflowID: admission.WorkflowID, SourceHash: admission.SourceHash,
		SourceRevision: admission.SourceRevision, ProgramHash: admission.ProgramHash, Status: string(record.Status()),
		QueuedAt: admission.QueuedAt.Format("2006-01-02T15:04:05.999999999Z07:00")}
}
