// Package workflow31 exposes the Yotta 3.1 application commands to Wails.
// It is a projection layer; execution and storage remain owned by Application.
package workflow31

import (
	"context"
	"errors"

	app31 "github.com/yottaapp/yotta/internal/application"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	run31 "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/workflow/authoring"
	"github.com/yottaapp/yotta/internal/workflow/schema"
	"github.com/yottaapp/yotta/internal/workflowstore"
)

type Service struct {
	application *app31.Application
	authoring   nodeauthoring.Snapshot
}

func NewService(application *app31.Application) (*Service, error) {
	if application == nil {
		return nil, errors.New("workflow service requires Application")
	}
	projection := application.AuthoringProjection()
	if !projection.Valid() {
		return nil, errors.New("workflow service requires trusted Authoring Projection")
	}
	return &Service{application: application, authoring: projection}, nil
}

type SourceView struct {
	WorkflowID string          `json:"workflowId"`
	Name       string          `json:"name"`
	Revision   int64           `json:"revision"`
	SourceHash artifact.Digest `json:"sourceHash"`
	SourceJSON string          `json:"sourceJson,omitempty"`
}

type CompileView struct {
	SourceHash  artifact.Digest     `json:"sourceHash,omitempty"`
	ProgramHash artifact.Digest     `json:"programHash,omitempty"`
	Diagnostics []schema.Diagnostic `json:"diagnostics"`
}

type RunView struct {
	RunID        string          `json:"runId"`
	Status       string          `json:"status"`
	Generation   uint64          `json:"generation"`
	RecordDigest artifact.Digest `json:"recordDigest"`
	ProgramHash  artifact.Digest `json:"programHash"`
	QueuedAt     string          `json:"queuedAt"`
	Failure      *FailureView    `json:"failure,omitempty"`
	Timeline     []TimelineEntry `json:"timeline"`
}

type FailureView struct {
	Code      string `json:"code"`
	Category  string `json:"category"`
	Retryable bool   `json:"retryable"`
	GraphID   string `json:"graphId,omitempty"`
	NodeID    string `json:"nodeId,omitempty"`
	Attempt   int    `json:"attempt,omitempty"`
}

type SummaryView struct {
	Code     string           `json:"code"`
	Counters map[string]int64 `json:"counters"`
}

type TimelineEntry struct {
	Sequence       uint64      `json:"sequence"`
	Kind           string      `json:"kind"`
	GraphPath      []string    `json:"graphPath"`
	NodeID         string      `json:"nodeId"`
	EffectID       string      `json:"effectId,omitempty"`
	Attempt        int         `json:"attempt"`
	Action         string      `json:"action,omitempty"`
	AttemptOutcome string      `json:"attemptOutcome,omitempty"`
	ActionOutcome  string      `json:"actionOutcome,omitempty"`
	OccurredAt     string      `json:"occurredAt"`
	ErrorCode      string      `json:"errorCode,omitempty"`
	StatusCode     string      `json:"statusCode,omitempty"`
	StatusCategory string      `json:"statusCategory,omitempty"`
	Summary        SummaryView `json:"summary"`
}

type StartRunView struct {
	SourceHash  artifact.Digest     `json:"sourceHash,omitempty"`
	ProgramHash artifact.Digest     `json:"programHash,omitempty"`
	Diagnostics []schema.Diagnostic `json:"diagnostics"`
	Run         *RunView            `json:"run,omitempty"`
}

type PatchView struct {
	Source         SourceView                `json:"source"`
	GeneratedNodes []authoring.GeneratedNode `json:"generatedNodes"`
}

func (s *Service) ApplyPatch(workflowID string, baseRevision int64, commands []authoring.Command) (PatchView, error) {
	result, err := s.application.ApplyPatch(context.Background(), authoring.PatchRequest{
		WorkflowID: workflowID, BaseRevision: baseRevision, Commands: commands,
	})
	if err != nil {
		return PatchView{}, err
	}
	view, err := sourceView(result.Source, true)
	if err != nil {
		return PatchView{}, err
	}
	return PatchView{Source: view, GeneratedNodes: result.GeneratedNodes}, nil
}

func (s *Service) CreateSource(name string) (SourceView, error) {
	snapshot, err := s.application.CreateSource(context.Background(), name)
	if err != nil {
		return SourceView{}, err
	}
	return sourceView(snapshot, true)
}

func (s *Service) GetSource(workflowID string) (SourceView, error) {
	snapshot, err := s.application.GetSource(workflowID)
	if err != nil {
		return SourceView{}, err
	}
	return sourceView(snapshot, true)
}

func (s *Service) ListSources() ([]SourceView, error) {
	snapshots := s.application.ListSources()
	result := make([]SourceView, 0, len(snapshots))
	for _, snapshot := range snapshots {
		view, err := sourceView(snapshot, false)
		if err != nil {
			return nil, err
		}
		result = append(result, view)
	}
	return result, nil
}

func (s *Service) CompileSource(workflowID string) (CompileView, error) {
	result, err := s.application.CompileSource(context.Background(), workflowID)
	view := CompileView{SourceHash: result.SourceHash, Diagnostics: append([]schema.Diagnostic(nil), result.Diagnostics...)}
	if program, ok := result.Program(); ok {
		view.ProgramHash = program.Hash()
	}
	return view, err
}

func (s *Service) PreviewRun(workflowID string) (app31.RunPreview, error) {
	return s.application.PreviewRun(context.Background(), workflowID)
}

func (s *Service) StartRun(workflowID string) (StartRunView, error) {
	result, err := s.application.StartRun(context.Background(), app31.StartRunRequest{WorkflowID: workflowID, Principal: "local-user"})
	view := StartRunView{
		SourceHash: result.SourceHash, ProgramHash: result.ProgramHash,
		Diagnostics: append([]schema.Diagnostic(nil), result.Diagnostics...),
	}
	if result.Record.Valid() {
		run := runView(result.Record)
		view.Run = &run
	}
	return view, err
}

func (s *Service) CancelRun(runID string) (RunView, error) {
	record, err := s.application.CancelRun(context.Background(), runID)
	if err != nil {
		return RunView{}, err
	}
	return runView(record), nil
}

func (s *Service) CancelAllRuns() error {
	return s.application.CancelAll(context.Background())
}

func (s *Service) GetRunTimeline(runID string) (RunView, error) {
	record, err := s.application.GetRun(runID)
	if err != nil {
		return RunView{}, err
	}
	return runView(record), nil
}

func (s *Service) GetCatalog() string { return string(s.application.CatalogArtifact()) }

func (s *Service) GetAuthoringProjection() string { return string(s.authoring.Bytes()) }

func sourceView(snapshot workflowstore.SourceSnapshot, includeSource bool) (SourceView, error) {
	document, diagnostics := schema.ParseSource(snapshot.Artifact())
	if len(diagnostics) != 0 {
		return SourceView{}, errors.New("stored Workflow Source failed strict reopen")
	}
	view := SourceView{
		WorkflowID: snapshot.WorkflowID(), Name: document.Workflow.Name,
		Revision: snapshot.Revision(), SourceHash: snapshot.Hash(),
	}
	if includeSource {
		view.SourceJSON = string(snapshot.Artifact())
	}
	return view, nil
}

func runView(record run31.Record) RunView {
	admission := record.Admission()
	view := RunView{
		RunID: admission.RunID, Status: string(record.Status()), Generation: record.Generation(), RecordDigest: record.Digest(),
		ProgramHash: admission.ProgramHash, QueuedAt: admission.QueuedAt.Format("2006-01-02T15:04:05.999999999Z07:00"),
		Timeline: timelineView(record.Journal()),
	}
	if failure, ok := record.Failure(); ok {
		view.Failure = &FailureView{
			Code: failure.Code, Category: failure.Category, Retryable: failure.Retryable,
			GraphID: failure.GraphID, NodeID: failure.NodeID, Attempt: failure.Attempt,
		}
	}
	return view
}

func timelineView(entries []run31.JournalEntry) []TimelineEntry {
	result := make([]TimelineEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, TimelineEntry{
			Sequence: entry.Sequence, Kind: string(entry.Kind), GraphPath: append([]string(nil), entry.GraphPath...),
			NodeID: entry.NodeID, EffectID: entry.EffectID, Attempt: entry.Attempt, Action: entry.Action,
			AttemptOutcome: string(entry.AttemptOutcome), ActionOutcome: string(entry.ActionOutcome),
			OccurredAt: entry.OccurredAt.Format("2006-01-02T15:04:05.999999999Z07:00"), ErrorCode: entry.ErrorCode,
			StatusCode: entry.StatusCode, StatusCategory: string(entry.StatusCategory),
			Summary: SummaryView{Code: entry.Summary.Code, Counters: entry.Summary.Counters},
		})
	}
	return result
}
