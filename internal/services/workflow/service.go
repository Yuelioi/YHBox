// Package workflow exposes the Yotta 3.1 application commands to Wails.
// It is a projection layer; execution and storage remain owned by Application.
package workflow

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	appcore "github.com/yottaapp/yotta/internal/application"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/workflow/authoring"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
	"github.com/yottaapp/yotta/internal/workflow/schema"
	"github.com/yottaapp/yotta/internal/workflowbundle"
	"github.com/yottaapp/yotta/internal/workflowstore"
)

type Service struct {
	application *appcore.Application
	authoring   nodeauthoring.Snapshot
	bundles     *workflowbundle.Manager
	references  ReferenceResolver
}

type ReferenceResolver func(workflowID string) []SourceReference

type Option func(*Service)

func WithReferenceResolver(resolver ReferenceResolver) Option {
	return func(service *Service) { service.references = resolver }
}

func WithBundleManager(manager *workflowbundle.Manager) Option {
	return func(service *Service) { service.bundles = manager }
}

func NewService(application *appcore.Application, options ...Option) (*Service, error) {
	if application == nil {
		return nil, errors.New("workflow service requires Application")
	}
	projection := application.AuthoringProjection()
	if !projection.Valid() {
		return nil, errors.New("workflow service requires trusted Authoring Projection")
	}
	service := &Service{application: application, authoring: projection}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	return service, nil
}

type SourceView struct {
	WorkflowID string          `json:"workflowId"`
	Name       string          `json:"name"`
	Revision   int64           `json:"revision"`
	SourceHash artifact.Digest `json:"sourceHash"`
	SourceJSON string          `json:"sourceJson,omitempty"`
}

type SourceQuery struct {
	Search   string `json:"search"`
	Sort     string `json:"sort"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

type SourcePage struct {
	Items    []SourceView `json:"items"`
	Total    int          `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
}

type SourceReference struct {
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	Label string `json:"label"`
}

type DeleteSourceRequest struct {
	WorkflowID string          `json:"workflowId"`
	Revision   int64           `json:"revision"`
	SourceHash artifact.Digest `json:"sourceHash"`
}

type DeleteSourcePreview struct {
	WorkflowID string            `json:"workflowId"`
	Name       string            `json:"name"`
	References []SourceReference `json:"references"`
}

type DeleteSourceResult struct {
	WorkflowID string            `json:"workflowId"`
	Deleted    bool              `json:"deleted"`
	References []SourceReference `json:"references"`
	Error      string            `json:"error,omitempty"`
}

type BundleInfoView struct {
	WorkflowID string          `json:"workflowId"`
	Name       string          `json:"name"`
	Revision   int64           `json:"revision"`
	SourceHash artifact.Digest `json:"sourceHash"`
	BlobCount  int             `json:"blobCount"`
	BlobBytes  int64           `json:"blobBytes"`
}

type BundleExportResult struct {
	WorkflowID string `json:"workflowId"`
	Path       string `json:"path,omitempty"`
	Exported   bool   `json:"exported"`
	Error      string `json:"error,omitempty"`
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
	SourceHash  artifact.Digest         `json:"sourceHash,omitempty"`
	ProgramHash artifact.Digest         `json:"programHash,omitempty"`
	Diagnostics []schema.Diagnostic     `json:"diagnostics"`
	Run         *RunView                `json:"run,omitempty"`
	Debug       *compiler.DebugSnapshot `json:"debug,omitempty"`
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

func (s *Service) QuerySources(query SourceQuery) (SourcePage, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		return SourcePage{}, errors.New("workflow source page size exceeds 100")
	}
	search := strings.ToLower(strings.TrimSpace(query.Search))
	views, err := s.ListSources()
	if err != nil {
		return SourcePage{}, err
	}
	filtered := views[:0]
	for _, view := range views {
		if search == "" || strings.Contains(strings.ToLower(view.Name), search) || strings.Contains(strings.ToLower(view.WorkflowID), search) {
			filtered = append(filtered, view)
		}
	}
	switch query.Sort {
	case "name_desc":
		sort.SliceStable(filtered, func(i, j int) bool { return strings.ToLower(filtered[i].Name) > strings.ToLower(filtered[j].Name) })
	case "revision_desc":
		sort.SliceStable(filtered, func(i, j int) bool {
			if filtered[i].Revision == filtered[j].Revision {
				return filtered[i].WorkflowID < filtered[j].WorkflowID
			}
			return filtered[i].Revision > filtered[j].Revision
		})
	case "", "name_asc":
		sort.SliceStable(filtered, func(i, j int) bool { return strings.ToLower(filtered[i].Name) < strings.ToLower(filtered[j].Name) })
	default:
		return SourcePage{}, fmt.Errorf("unsupported workflow source sort %q", query.Sort)
	}
	total := len(filtered)
	start := (query.Page - 1) * query.PageSize
	if start > total {
		start = total
	}
	end := min(start+query.PageSize, total)
	items := append([]SourceView(nil), filtered[start:end]...)
	return SourcePage{Items: items, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}

func (s *Service) InspectSourceBundle(archivePath string) (BundleInfoView, error) {
	if s.bundles == nil {
		return BundleInfoView{}, errors.New("workflow source portability is unavailable")
	}
	info, err := s.bundles.Inspect(context.Background(), archivePath)
	if err != nil {
		return BundleInfoView{}, err
	}
	return bundleInfoView(info), nil
}

func (s *Service) ImportSourceBundle(archivePath string) (SourceView, error) {
	if s.bundles == nil {
		return SourceView{}, errors.New("workflow source portability is unavailable")
	}
	result, err := s.bundles.Import(context.Background(), workflowbundle.ImportRequest{Path: archivePath, Mode: workflowbundle.ImportCopy})
	if err != nil {
		return SourceView{}, err
	}
	return sourceView(result.Source, false)
}

func (s *Service) ReplaceSourceFromBundle(archivePath, targetWorkflowID string, expectedRevision int64, expectedSourceHash artifact.Digest) (SourceView, error) {
	if s.bundles == nil {
		return SourceView{}, errors.New("workflow source portability is unavailable")
	}
	result, err := s.bundles.Import(context.Background(), workflowbundle.ImportRequest{
		Path: archivePath, Mode: workflowbundle.ImportReplace, TargetWorkflowID: targetWorkflowID,
		ExpectedRevision: expectedRevision, ExpectedSourceHash: expectedSourceHash,
	})
	if err != nil {
		return SourceView{}, err
	}
	return sourceView(result.Source, false)
}

func (s *Service) ExportSourceBundle(workflowID, destination string) (BundleExportResult, error) {
	if s.bundles == nil {
		return BundleExportResult{}, errors.New("workflow source portability is unavailable")
	}
	result, err := s.bundles.Export(context.Background(), workflowID, destination)
	if err != nil {
		return BundleExportResult{}, err
	}
	return BundleExportResult{WorkflowID: workflowID, Path: result.Path, Exported: true}, nil
}

func (s *Service) ExportSourceBundles(workflowIDs []string, directory string) []BundleExportResult {
	results := make([]BundleExportResult, 0, len(workflowIDs))
	seen := make(map[string]struct{}, len(workflowIDs))
	for _, workflowID := range workflowIDs {
		result := BundleExportResult{WorkflowID: workflowID}
		if s.bundles == nil {
			result.Error = "workflow source portability is unavailable"
		} else if strings.TrimSpace(directory) == "" {
			result.Error = "workflow source export directory is required"
		} else if _, duplicate := seen[workflowID]; duplicate {
			result.Error = "duplicate workflow source export request"
		} else {
			seen[workflowID] = struct{}{}
			destination := filepath.Join(directory, workflowID+workflowbundle.Extension)
			if _, err := os.Lstat(destination); err == nil {
				result.Error = "destination already exists"
			} else if !errors.Is(err, os.ErrNotExist) {
				result.Error = err.Error()
			} else if exported, err := s.bundles.Export(context.Background(), workflowID, destination); err != nil {
				result.Error = err.Error()
			} else {
				result.Exported = true
				result.Path = exported.Path
			}
		}
		results = append(results, result)
	}
	return results
}

func (s *Service) PreviewDeleteSources(workflowIDs []string) ([]DeleteSourcePreview, error) {
	result := make([]DeleteSourcePreview, 0, len(workflowIDs))
	seen := make(map[string]struct{}, len(workflowIDs))
	for _, workflowID := range workflowIDs {
		if _, duplicate := seen[workflowID]; duplicate {
			continue
		}
		seen[workflowID] = struct{}{}
		snapshot, err := s.application.GetSource(workflowID)
		if err != nil {
			return nil, err
		}
		view, err := sourceView(snapshot, false)
		if err != nil {
			return nil, err
		}
		result = append(result, DeleteSourcePreview{WorkflowID: workflowID, Name: view.Name, References: s.sourceReferences(workflowID)})
	}
	return result, nil
}

func (s *Service) DeleteSources(requests []DeleteSourceRequest) []DeleteSourceResult {
	results := make([]DeleteSourceResult, 0, len(requests))
	seen := make(map[string]struct{}, len(requests))
	for _, request := range requests {
		result := DeleteSourceResult{WorkflowID: request.WorkflowID}
		if _, duplicate := seen[request.WorkflowID]; duplicate {
			result.Error = "duplicate workflow source delete request"
			results = append(results, result)
			continue
		}
		seen[request.WorkflowID] = struct{}{}
		result.References = s.sourceReferences(request.WorkflowID)
		if len(result.References) != 0 {
			result.Error = "workflow source is referenced"
			results = append(results, result)
			continue
		}
		if err := s.application.DeleteSource(context.Background(), request.WorkflowID, request.Revision, request.SourceHash); err != nil {
			result.Error = err.Error()
		} else {
			result.Deleted = true
		}
		results = append(results, result)
	}
	return results
}

func (s *Service) sourceReferences(workflowID string) []SourceReference {
	references := make([]SourceReference, 0)
	for _, runID := range s.application.ActiveSourceRuns(workflowID) {
		references = append(references, SourceReference{Kind: "active_run", ID: runID, Label: runID})
	}
	if s.references != nil {
		references = append(references, s.references(workflowID)...)
	}
	sort.SliceStable(references, func(i, j int) bool {
		if references[i].Kind == references[j].Kind {
			return references[i].ID < references[j].ID
		}
		return references[i].Kind < references[j].Kind
	})
	return references
}

func (s *Service) CompileSource(workflowID string) (CompileView, error) {
	result, err := s.application.CompileSource(context.Background(), workflowID)
	view := CompileView{SourceHash: result.SourceHash, Diagnostics: append([]schema.Diagnostic(nil), result.Diagnostics...)}
	if program, ok := result.Program(); ok {
		view.ProgramHash = program.Hash()
	}
	return view, err
}

func (s *Service) PreviewRun(workflowID string) (appcore.RunPreview, error) {
	return s.application.PreviewRun(context.Background(), workflowID)
}

func (s *Service) StartRun(workflowID string) (StartRunView, error) {
	result, err := s.application.StartRun(context.Background(), appcore.StartRunRequest{WorkflowID: workflowID, Principal: "local-user"})
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

func (s *Service) StartDebugRun(workflowID string, breakpoints []compiler.DebugBreakpoint) (StartRunView, error) {
	result, err := s.application.StartDebugRun(context.Background(), appcore.StartRunRequest{WorkflowID: workflowID, Principal: "local-user"}, breakpoints)
	view := StartRunView{
		SourceHash: result.SourceHash, ProgramHash: result.ProgramHash,
		Diagnostics: append([]schema.Diagnostic(nil), result.Diagnostics...),
	}
	if result.Record.Valid() {
		run := runView(result.Record)
		view.Run = &run
		if snapshot, snapshotErr := s.application.GetDebugSnapshot(run.RunID); snapshotErr == nil {
			view.Debug = &snapshot
		}
	}
	return view, err
}

func (s *Service) GetDebugSnapshot(runID string) (compiler.DebugSnapshot, error) {
	return s.application.GetDebugSnapshot(runID)
}

func (s *Service) ControlDebugRun(runID, action string) (compiler.DebugSnapshot, error) {
	return s.application.ControlDebugRun(context.Background(), runID, appcore.DebugAction(action))
}

func (s *Service) SetDebugBreakpoints(runID string, breakpoints []compiler.DebugBreakpoint) (compiler.DebugSnapshot, error) {
	return s.application.SetDebugBreakpoints(context.Background(), runID, breakpoints)
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

func bundleInfoView(info workflowbundle.Info) BundleInfoView {
	return BundleInfoView{
		WorkflowID: info.WorkflowID, Name: info.Name, Revision: info.Revision,
		SourceHash: info.SourceHash, BlobCount: info.BlobCount, BlobBytes: info.BlobBytes,
	}
}

func runView(record run.Record) RunView {
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

func timelineView(entries []run.JournalEntry) []TimelineEntry {
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
