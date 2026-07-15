// Package workflow31 exposes the Yotta 3.1 application commands to Wails.
// It is a projection layer; execution and storage remain owned by Application.
package workflow31

import (
	"context"
	"errors"

	app31 "github.com/yottaapp/yotta/internal/application"
	"github.com/yottaapp/yotta/internal/artifact"
	run31 "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

type Service struct{ application *app31.Application }

func NewService(application *app31.Application) (*Service, error) {
	if application == nil {
		return nil, errors.New("workflow service requires Application")
	}
	return &Service{application: application}, nil
}

type SourceView struct {
	WorkflowID string          `json:"workflowId"`
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
	RunID        string               `json:"runId"`
	Status       run31.Status         `json:"status"`
	Generation   uint64               `json:"generation"`
	RecordDigest artifact.Digest      `json:"recordDigest"`
	ProgramHash  artifact.Digest      `json:"programHash"`
	QueuedAt     string               `json:"queuedAt"`
	Failure      *run31.RunError      `json:"failure,omitempty"`
	Timeline     []run31.JournalEntry `json:"timeline"`
}

type StartRunView struct {
	SourceHash  artifact.Digest     `json:"sourceHash,omitempty"`
	ProgramHash artifact.Digest     `json:"programHash,omitempty"`
	Diagnostics []schema.Diagnostic `json:"diagnostics"`
	Run         *RunView            `json:"run,omitempty"`
}

func (s *Service) SaveSource(sourceJSON string, baseRevision int64) (SourceView, error) {
	snapshot, err := s.application.SaveSource(context.Background(), []byte(sourceJSON), baseRevision)
	if err != nil {
		return SourceView{}, err
	}
	return sourceView(snapshot.WorkflowID(), snapshot.Revision(), snapshot.Hash(), snapshot.Artifact()), nil
}

func (s *Service) GetSource(workflowID string) (SourceView, error) {
	snapshot, err := s.application.GetSource(workflowID)
	if err != nil {
		return SourceView{}, err
	}
	return sourceView(snapshot.WorkflowID(), snapshot.Revision(), snapshot.Hash(), snapshot.Artifact()), nil
}

func (s *Service) ListSources() []SourceView {
	snapshots := s.application.ListSources()
	result := make([]SourceView, 0, len(snapshots))
	for _, snapshot := range snapshots {
		result = append(result, sourceView(snapshot.WorkflowID(), snapshot.Revision(), snapshot.Hash(), nil))
	}
	return result
}

func (s *Service) CompileDraft(sourceJSON string) (CompileView, error) {
	result, err := s.application.CompileDraft(context.Background(), []byte(sourceJSON))
	view := CompileView{SourceHash: result.SourceHash, Diagnostics: append([]schema.Diagnostic(nil), result.Diagnostics...)}
	if program, ok := result.Program(); ok {
		view.ProgramHash = program.Hash()
	}
	return view, err
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

func (s *Service) GetRunTimeline(runID string) (RunView, error) {
	record, err := s.application.GetRun(runID)
	if err != nil {
		return RunView{}, err
	}
	return runView(record), nil
}

func (s *Service) GetCatalog() string { return string(s.application.CatalogArtifact()) }

func sourceView(workflowID string, revision int64, hash artifact.Digest, raw []byte) SourceView {
	return SourceView{WorkflowID: workflowID, Revision: revision, SourceHash: hash, SourceJSON: string(raw)}
}

func runView(record run31.Record) RunView {
	admission := record.Admission()
	view := RunView{
		RunID: admission.RunID, Status: record.Status(), Generation: record.Generation(), RecordDigest: record.Digest(),
		ProgramHash: admission.ProgramHash, QueuedAt: admission.QueuedAt.Format("2006-01-02T15:04:05.999999999Z07:00"),
		Timeline: record.Journal(),
	}
	if failure, ok := record.Failure(); ok {
		view.Failure = &failure
	}
	return view
}
