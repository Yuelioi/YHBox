// Package workflow exposes the Yotta application commands to Wails.
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
	"time"

	appcore "github.com/yottaapp/yotta/internal/application"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/workflow/authoring"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
	"github.com/yottaapp/yotta/internal/workflow/schema"
	"github.com/yottaapp/yotta/internal/workflowbundle"
	"github.com/yottaapp/yotta/internal/workflowinstallation"
	"github.com/yottaapp/yotta/internal/workflowstore"
)

type Service struct {
	application   *appcore.Application
	authoring     nodeauthoring.Snapshot
	bundles       *workflowbundle.Manager
	references    ReferenceResolver
	installations InstallationRuntime
}

type ReferenceResolver func(workflowID string) []SourceReference

type Option func(*Service)

type InstallationRuntime interface {
	ListWorkflowInstallations(context.Context) ([]workflowinstallation.InstallationRecord, error)
	WorkflowInstallationReadiness(context.Context, string) (workflowinstallation.ReadinessReport, error)
	WorkflowInstallationSettings(context.Context, string) (workflowinstallation.SettingsSnapshot, error)
	UpdateWorkflowInstallationTargetProfile(context.Context, string, int64, string, []byte, string) (workflowinstallation.SettingsSnapshot, error)
	UpdateWorkflowInstallationCredentialBinding(context.Context, string, int64, string, string) (workflowinstallation.SettingsSnapshot, error)
	GrantWorkflowInstallationConsent(context.Context, string, workflowinstallation.ExecutionScope) (workflowinstallation.ReadinessReport, error)
	StartInstallationRun(context.Context, string, workflowinstallation.ExecutionScope) (appcore.StartRunResult, error)
}

func WithReferenceResolver(resolver ReferenceResolver) Option {
	return func(service *Service) { service.references = resolver }
}

func WithBundleManager(manager *workflowbundle.Manager) Option {
	return func(service *Service) { service.bundles = manager }
}

func WithInstallationRuntime(runtime InstallationRuntime) Option {
	return func(service *Service) { service.installations = runtime }
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
	WorkflowID  string          `json:"workflowId"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Category    string          `json:"category,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
	CreatedAt   string          `json:"createdAt,omitempty"`
	UpdatedAt   string          `json:"updatedAt,omitempty"`
	NodeCount   int             `json:"nodeCount"`
	Revision    int64           `json:"revision"`
	SourceHash  artifact.Digest `json:"sourceHash"`
	SourceJSON  string          `json:"sourceJson,omitempty"`
}

type InstallationView struct {
	InstallationID string          `json:"installationId"`
	ReleaseID      artifact.Digest `json:"releaseId"`
	Name           string          `json:"name"`
	Lifecycle      string          `json:"lifecycle"`
	CreatedAt      string          `json:"createdAt"`
	UpdatedAt      string          `json:"updatedAt"`
}

type ReadinessBlockerView struct {
	Kind          string   `json:"kind"`
	RequirementID string   `json:"requirementId"`
	Expected      string   `json:"expected"`
	Blocks        []string `json:"blocks"`
	Action        string   `json:"action"`
}

type InstallationReadinessView struct {
	InstallationID           string                 `json:"installationId"`
	ReleaseID                artifact.Digest        `json:"releaseId"`
	Lifecycle                string                 `json:"lifecycle"`
	LifecycleAllowsExecution bool                   `json:"lifecycleAllowsExecution"`
	RunAllowed               bool                   `json:"runAllowed"`
	ScheduleAllowed          bool                   `json:"scheduleAllowed"`
	Blockers                 []ReadinessBlockerView `json:"blockers"`
}

type TargetDiscoveryHintView struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type InstallationTargetProfileView struct {
	DefinitionID         string                    `json:"definitionId"`
	Name                 string                    `json:"name"`
	Description          string                    `json:"description"`
	TargetKind           string                    `json:"targetKind"`
	AdapterKind          string                    `json:"adapterKind"`
	ProfileVersion       string                    `json:"profileVersion"`
	SettingsJSON         string                    `json:"settingsJson"`
	TargetInstallationID string                    `json:"targetInstallationId"`
	DiscoveryHints       []TargetDiscoveryHintView `json:"discoveryHints"`
}

type InstallationCredentialRequirementView struct {
	Slot       string                                `json:"slot"`
	Kind       string                                `json:"kind"`
	Purpose    string                                `json:"purpose"`
	BindingID  string                                `json:"bindingId"`
	Candidates []InstallationCredentialCandidateView `json:"candidates"`
}

type InstallationCredentialCandidateView struct {
	BindingID string `json:"bindingId"`
	Label     string `json:"label"`
	Available bool   `json:"available"`
}

type InstallationSettingsView struct {
	InstallationID string                                  `json:"installationId"`
	Generation     int64                                   `json:"generation"`
	UpdatedAt      string                                  `json:"updatedAt"`
	Targets        []InstallationTargetProfileView         `json:"targets"`
	Credentials    []InstallationCredentialRequirementView `json:"credentials"`
}

type SourceQuery struct {
	Search       string   `json:"search"`
	Category     string   `json:"category"`
	Tags         []string `json:"tags"`
	CreatedSince string   `json:"createdSince"`
	UpdatedSince string   `json:"updatedSince"`
	Sort         string   `json:"sort"`
	Page         int      `json:"page"`
	PageSize     int      `json:"pageSize"`
}

type FacetValue struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

type SourcePage struct {
	Items      []SourceView `json:"items"`
	Total      int          `json:"total"`
	Page       int          `json:"page"`
	PageSize   int          `json:"pageSize"`
	Categories []FacetValue `json:"categories"`
	Tags       []FacetValue `json:"tags"`
}

type CreateSourceRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
}

type UpdateSourceMetadataRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
}

type BatchUpdateSourceMetadataRequest struct {
	WorkflowID   string   `json:"workflowId"`
	BaseRevision int64    `json:"baseRevision"`
	Category     string   `json:"category"`
	Tags         []string `json:"tags"`
}

type BatchUpdateSourceMetadataResult struct {
	WorkflowID string `json:"workflowId"`
	Updated    bool   `json:"updated"`
	Error      string `json:"error,omitempty"`
}

type SourceRecoveryView struct {
	RecoveryID   artifact.Digest `json:"recoveryId"`
	OriginalName string          `json:"originalName"`
	Reason       string          `json:"reason"`
	SourceJSON   string          `json:"sourceJson"`
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
	RunID         string          `json:"runId"`
	Status        string          `json:"status"`
	Generation    uint64          `json:"generation"`
	RecordDigest  artifact.Digest `json:"recordDigest"`
	ProgramHash   artifact.Digest `json:"programHash"`
	QueuedAt      string          `json:"queuedAt"`
	Failure       *FailureView    `json:"failure,omitempty"`
	Timeline      []TimelineEntry `json:"timeline"`
	TimelinePage  int             `json:"timelinePage"`
	TimelinePages int             `json:"timelinePages"`
	TimelineTotal int             `json:"timelineTotal"`
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

func (s *Service) CreateSourceWithMetadata(request CreateSourceRequest) (SourceView, error) {
	snapshot, err := s.application.CreateSourceWithMetadata(context.Background(), authoring.WorkflowMetadata{
		Name: request.Name, Description: request.Description, Category: request.Category, Tags: request.Tags,
	})
	if err != nil {
		return SourceView{}, err
	}
	return sourceView(snapshot, true)
}

func (s *Service) UpdateSourceMetadata(workflowID string, baseRevision int64, request UpdateSourceMetadataRequest) (SourceView, error) {
	patch, err := s.ApplyPatch(workflowID, baseRevision, []authoring.Command{{
		Kind: authoring.CommandUpdateWorkflowMetadata,
		UpdateWorkflowMetadata: &authoring.UpdateWorkflowMetadataCommand{
			Name: request.Name, Description: request.Description, Category: request.Category, Tags: request.Tags,
		},
	}})
	if err != nil {
		return SourceView{}, err
	}
	return patch.Source, nil
}

func (s *Service) BatchUpdateSourceMetadata(requests []BatchUpdateSourceMetadataRequest) []BatchUpdateSourceMetadataResult {
	results := make([]BatchUpdateSourceMetadataResult, 0, len(requests))
	seen := make(map[string]struct{}, len(requests))
	for _, request := range requests {
		result := BatchUpdateSourceMetadataResult{WorkflowID: request.WorkflowID}
		if _, duplicate := seen[request.WorkflowID]; duplicate {
			result.Error = "duplicate workflow source metadata request"
			results = append(results, result)
			continue
		}
		seen[request.WorkflowID] = struct{}{}
		current, err := s.GetSource(request.WorkflowID)
		if err == nil {
			_, err = s.UpdateSourceMetadata(request.WorkflowID, request.BaseRevision, UpdateSourceMetadataRequest{
				Name:        current.Name,
				Description: current.Description,
				Category:    request.Category,
				Tags:        request.Tags,
			})
		}
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Updated = true
		}
		results = append(results, result)
	}
	return results
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

func (s *Service) ListSourceRecoveries() []SourceRecoveryView {
	recoveries := s.application.ListSourceRecoveries()
	result := make([]SourceRecoveryView, 0, len(recoveries))
	for _, recovery := range recoveries {
		result = append(result, SourceRecoveryView{
			RecoveryID: recovery.ID, OriginalName: recovery.OriginalName,
			Reason: recovery.Reason, SourceJSON: string(recovery.Artifact()),
		})
	}
	return result
}

func (s *Service) RepairSourceRecovery(recoveryID artifact.Digest, sourceJSON string) (SourceView, error) {
	snapshot, err := s.application.RepairSourceRecovery(context.Background(), recoveryID, []byte(sourceJSON))
	if err != nil {
		return SourceView{}, err
	}
	return sourceView(snapshot, false)
}

func (s *Service) DeleteSourceRecovery(recoveryID artifact.Digest) error {
	return s.application.DeleteSourceRecovery(context.Background(), recoveryID)
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
	if len([]rune(query.Search)) > 200 || len([]rune(query.Category)) > 128 || len(query.Tags) > 16 ||
		len(query.CreatedSince) > 64 || len(query.UpdatedSince) > 64 {
		return SourcePage{}, errors.New("workflow source filter budget exceeded")
	}
	createdSince, err := parseSourceFilterTime("createdSince", query.CreatedSince)
	if err != nil {
		return SourcePage{}, err
	}
	updatedSince, err := parseSourceFilterTime("updatedSince", query.UpdatedSince)
	if err != nil {
		return SourcePage{}, err
	}
	search := strings.ToLower(strings.TrimSpace(query.Search))
	category := strings.ToLower(strings.TrimSpace(query.Category))
	wantedTags := normalizedSourceTags(query.Tags)
	views, err := s.ListSources()
	if err != nil {
		return SourcePage{}, err
	}
	categories, tags := sourceFacets(views)
	filtered := make([]SourceView, 0, len(views))
	for _, view := range views {
		if category != "" && strings.ToLower(view.Category) != category {
			continue
		}
		if !containsEverySourceTag(view.Tags, wantedTags) {
			continue
		}
		if !sourceTimeMatches(view.CreatedAt, createdSince) || !sourceTimeMatches(view.UpdatedAt, updatedSince) {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(strings.Join(append([]string{
			view.Name, view.Description, view.Category, view.WorkflowID,
		}, view.Tags...), " ")), search) {
			continue
		}
		filtered = append(filtered, view)
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
	case "nodes_desc":
		sort.SliceStable(filtered, func(i, j int) bool {
			if filtered[i].NodeCount == filtered[j].NodeCount {
				return strings.ToLower(filtered[i].Name) < strings.ToLower(filtered[j].Name)
			}
			return filtered[i].NodeCount > filtered[j].NodeCount
		})
	case "created_desc":
		sortSourceViewsByTime(filtered, func(view SourceView) string { return view.CreatedAt })
	case "updated_desc":
		sortSourceViewsByTime(filtered, func(view SourceView) string { return view.UpdatedAt })
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
	return SourcePage{
		Items: items, Total: total, Page: query.Page, PageSize: query.PageSize,
		Categories: categories, Tags: tags,
	}, nil
}

func parseSourceFilterTime(field, value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid workflow source %s filter", field)
	}
	return parsed, nil
}

func sourceTimeMatches(value string, since time.Time) bool {
	if since.IsZero() {
		return true
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	return err == nil && !parsed.Before(since)
}

func sortSourceViewsByTime(views []SourceView, value func(SourceView) string) {
	sort.SliceStable(views, func(i, j int) bool {
		left, leftErr := time.Parse(time.RFC3339Nano, value(views[i]))
		right, rightErr := time.Parse(time.RFC3339Nano, value(views[j]))
		if leftErr != nil || rightErr != nil {
			if leftErr != nil && rightErr != nil {
				return strings.ToLower(views[i].Name) < strings.ToLower(views[j].Name)
			}
			return leftErr == nil
		}
		if left.Equal(right) {
			return strings.ToLower(views[i].Name) < strings.ToLower(views[j].Name)
		}
		return left.After(right)
	})
}

func sourceFacets(views []SourceView) ([]FacetValue, []FacetValue) {
	categoryCounts := make(map[string]int)
	tagCounts := make(map[string]int)
	categoryLabels := make(map[string]string)
	tagLabels := make(map[string]string)
	for _, view := range views {
		if category := strings.TrimSpace(view.Category); category != "" {
			key := strings.ToLower(category)
			categoryCounts[key]++
			if categoryLabels[key] == "" {
				categoryLabels[key] = category
			}
		}
		seen := make(map[string]struct{}, len(view.Tags))
		for _, raw := range view.Tags {
			tag := strings.TrimSpace(raw)
			key := strings.ToLower(tag)
			if key == "" {
				continue
			}
			if _, duplicate := seen[key]; duplicate {
				continue
			}
			seen[key] = struct{}{}
			tagCounts[key]++
			if tagLabels[key] == "" {
				tagLabels[key] = tag
			}
		}
	}
	return sortedFacetValues(categoryCounts, categoryLabels), sortedFacetValues(tagCounts, tagLabels)
}

func sortedFacetValues(counts map[string]int, labels map[string]string) []FacetValue {
	values := make([]FacetValue, 0, len(counts))
	for key, count := range counts {
		values = append(values, FacetValue{Value: labels[key], Count: count})
	}
	sort.Slice(values, func(i, j int) bool { return strings.ToLower(values[i].Value) < strings.ToLower(values[j].Value) })
	return values
}

func normalizedSourceTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	result := make([]string, 0, len(tags))
	for _, raw := range tags {
		tag := strings.ToLower(strings.TrimSpace(raw))
		if tag == "" {
			continue
		}
		if _, duplicate := seen[tag]; duplicate {
			continue
		}
		seen[tag] = struct{}{}
		result = append(result, tag)
	}
	return result
}

func containsEverySourceTag(tags, wanted []string) bool {
	if len(wanted) == 0 {
		return true
	}
	available := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		available[strings.ToLower(strings.TrimSpace(tag))] = struct{}{}
	}
	for _, tag := range wanted {
		if _, ok := available[tag]; !ok {
			return false
		}
	}
	return true
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

func (s *Service) ListInstallations() ([]InstallationView, error) {
	if s.installations == nil {
		return nil, errors.New("Workflow Installation runtime is unavailable")
	}
	records, err := s.installations.ListWorkflowInstallations(context.Background())
	if err != nil {
		return nil, err
	}
	result := make([]InstallationView, 0, len(records))
	for _, record := range records {
		result = append(result, InstallationView{
			InstallationID: record.ID, ReleaseID: record.ReleaseID, Name: record.Name,
			Lifecycle: string(record.Lifecycle),
			CreatedAt: record.CreatedAt.Format(time.RFC3339Nano),
			UpdatedAt: record.UpdatedAt.Format(time.RFC3339Nano),
		})
	}
	return result, nil
}

func (s *Service) GetInstallationReadiness(installationID string) (InstallationReadinessView, error) {
	if s.installations == nil {
		return InstallationReadinessView{}, errors.New("Workflow Installation runtime is unavailable")
	}
	report, err := s.installations.WorkflowInstallationReadiness(context.Background(), installationID)
	if err != nil {
		return InstallationReadinessView{}, err
	}
	return installationReadinessView(report), nil
}

func (s *Service) GetInstallationSettings(installationID string) (InstallationSettingsView, error) {
	if s.installations == nil {
		return InstallationSettingsView{}, errors.New("Workflow Installation runtime is unavailable")
	}
	snapshot, err := s.installations.WorkflowInstallationSettings(context.Background(), installationID)
	if err != nil {
		return InstallationSettingsView{}, err
	}
	return installationSettingsView(snapshot), nil
}

func (s *Service) UpdateInstallationTargetProfile(
	installationID string,
	expectedGeneration int64,
	definitionID string,
	settingsJSON string,
	targetInstallationID string,
) (InstallationSettingsView, error) {
	if s.installations == nil {
		return InstallationSettingsView{}, errors.New("Workflow Installation runtime is unavailable")
	}
	snapshot, err := s.installations.UpdateWorkflowInstallationTargetProfile(
		context.Background(), installationID, expectedGeneration, definitionID,
		[]byte(settingsJSON), targetInstallationID,
	)
	if err != nil {
		return InstallationSettingsView{}, err
	}
	return installationSettingsView(snapshot), nil
}

func (s *Service) UpdateInstallationCredentialBinding(
	installationID string,
	expectedGeneration int64,
	requirementSlot string,
	credentialBindingID string,
) (InstallationSettingsView, error) {
	if s.installations == nil {
		return InstallationSettingsView{}, errors.New("Workflow Installation runtime is unavailable")
	}
	snapshot, err := s.installations.UpdateWorkflowInstallationCredentialBinding(
		context.Background(), installationID, expectedGeneration,
		requirementSlot, credentialBindingID,
	)
	if err != nil {
		return InstallationSettingsView{}, err
	}
	return installationSettingsView(snapshot), nil
}

func (s *Service) GrantInstallationConsent(
	installationID string,
	scope string,
) (InstallationReadinessView, error) {
	if s.installations == nil {
		return InstallationReadinessView{}, errors.New("Workflow Installation runtime is unavailable")
	}
	executionScope := workflowinstallation.ExecutionScope(scope)
	if executionScope != workflowinstallation.ScopeRun && executionScope != workflowinstallation.ScopeSchedule {
		return InstallationReadinessView{}, errors.New("Workflow Installation consent scope is invalid")
	}
	report, err := s.installations.GrantWorkflowInstallationConsent(
		context.Background(), installationID, executionScope,
	)
	if err != nil {
		return InstallationReadinessView{}, err
	}
	return installationReadinessView(report), nil
}

func (s *Service) StartInstallationRun(installationID string) (StartRunView, error) {
	if s.installations == nil {
		return StartRunView{}, errors.New("Workflow Installation runtime is unavailable")
	}
	result, err := s.installations.StartInstallationRun(
		context.Background(), installationID, workflowinstallation.ScopeRun,
	)
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

func installationReadinessView(report workflowinstallation.ReadinessReport) InstallationReadinessView {
	view := InstallationReadinessView{
		InstallationID: report.InstallationID, ReleaseID: report.ReleaseID,
		Lifecycle: string(report.Lifecycle), LifecycleAllowsExecution: report.LifecycleAllowsExecution,
		RunAllowed: report.RunAllowed, ScheduleAllowed: report.ScheduleAllowed,
		Blockers: make([]ReadinessBlockerView, 0, len(report.Blockers)),
	}
	for _, blocker := range report.Blockers {
		blocks := make([]string, 0, len(blocker.Blocks))
		for _, scope := range blocker.Blocks {
			blocks = append(blocks, string(scope))
		}
		view.Blockers = append(view.Blockers, ReadinessBlockerView{
			Kind: string(blocker.Kind), RequirementID: blocker.RequirementID,
			Expected: blocker.Expected, Blocks: blocks, Action: string(blocker.Action.Kind),
		})
	}
	return view
}

func installationSettingsView(snapshot workflowinstallation.SettingsSnapshot) InstallationSettingsView {
	view := InstallationSettingsView{
		InstallationID: snapshot.Configuration.InstallationID,
		Generation:     snapshot.Configuration.Generation,
		UpdatedAt:      snapshot.Configuration.UpdatedAt.Format(time.RFC3339Nano),
		Targets:        make([]InstallationTargetProfileView, 0, len(snapshot.TargetDefinitions)),
		Credentials:    make([]InstallationCredentialRequirementView, 0, len(snapshot.CredentialRequirements)),
	}
	for _, definition := range snapshot.TargetDefinitions {
		profile := snapshot.Configuration.TargetProfiles[definition.ID]
		target := InstallationTargetProfileView{
			DefinitionID: definition.ID, Name: definition.Name, Description: definition.Description,
			TargetKind: definition.TargetKind, AdapterKind: definition.AdapterKind,
			ProfileVersion: definition.ProfileVersion, SettingsJSON: string(profile.Settings),
			TargetInstallationID: profile.TargetInstallationID,
			DiscoveryHints:       make([]TargetDiscoveryHintView, 0, len(definition.DiscoveryHints)),
		}
		for _, hint := range definition.DiscoveryHints {
			target.DiscoveryHints = append(target.DiscoveryHints, TargetDiscoveryHintView{
				Kind: hint.Kind, Value: hint.Value,
			})
		}
		view.Targets = append(view.Targets, target)
	}
	for _, requirement := range snapshot.CredentialRequirements {
		credential := InstallationCredentialRequirementView{
			Slot: requirement.Slot, Kind: requirement.Kind, Purpose: requirement.Purpose,
			BindingID:  snapshot.Configuration.CredentialBindings[requirement.Slot],
			Candidates: []InstallationCredentialCandidateView{},
		}
		for _, candidate := range snapshot.Credentials {
			if candidate.Kind != requirement.Kind {
				continue
			}
			credential.Candidates = append(credential.Candidates, InstallationCredentialCandidateView{
				BindingID: candidate.CredentialBindingID,
				Label:     candidate.Label,
				Available: candidate.Available,
			})
		}
		view.Credentials = append(view.Credentials, credential)
	}
	return view
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

func (s *Service) GetRunTimelinePage(runID string, page, pageSize int) (RunView, error) {
	timeline, err := s.application.GetRunTimelinePage(context.Background(), runID, page, pageSize)
	if err != nil {
		return RunView{}, err
	}
	return runTimelinePageView(timeline), nil
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
		Description: document.Workflow.Description, Category: document.Workflow.Category,
		Tags: append([]string(nil), document.Workflow.Tags...), CreatedAt: document.Workflow.CreatedAt,
		UpdatedAt: document.Workflow.UpdatedAt, NodeCount: sourceNodeCount(document),
		Revision: snapshot.Revision(), SourceHash: snapshot.Hash(),
	}
	if includeSource {
		view.SourceJSON = string(snapshot.Artifact())
	}
	return view, nil
}

func sourceNodeCount(source schema.WorkflowSource) int {
	total := 0
	for _, graph := range source.Graphs {
		total += len(graph.Nodes) + len(graph.Calls)
	}
	return total
}

func bundleInfoView(info workflowbundle.Info) BundleInfoView {
	return BundleInfoView{
		WorkflowID: info.WorkflowID, Name: info.Name, Revision: info.Revision,
		SourceHash: info.SourceHash, BlobCount: info.BlobCount, BlobBytes: info.BlobBytes,
	}
}

func runView(record run.Record) RunView {
	return runViewPage(record, 1, 200)
}

func runViewPage(record run.Record, page, pageSize int) RunView {
	admission := record.Admission()
	entries := record.Journal()
	pageEntries, currentPage, pages := timelinePage(entries, page, pageSize)
	view := RunView{
		RunID: admission.RunID, Status: string(record.Status()), Generation: record.Generation(), RecordDigest: record.Digest(),
		ProgramHash: admission.ProgramHash, QueuedAt: admission.QueuedAt.Format("2006-01-02T15:04:05.999999999Z07:00"),
		Timeline: timelineView(pageEntries), TimelinePage: currentPage, TimelinePages: pages, TimelineTotal: len(entries),
	}
	if failure, ok := record.Failure(); ok {
		view.Failure = &FailureView{
			Code: failure.Code, Category: failure.Category, Retryable: failure.Retryable,
			GraphID: failure.GraphID, NodeID: failure.NodeID, Attempt: failure.Attempt,
		}
	}
	return view
}

func runTimelinePageView(timeline run.TimelinePage) RunView {
	summary := timeline.Summary
	admission := summary.Admission
	view := RunView{
		RunID: admission.RunID, Status: string(summary.Status),
		Generation: summary.Generation, RecordDigest: summary.Digest,
		ProgramHash: admission.ProgramHash,
		QueuedAt:    admission.QueuedAt.Format("2006-01-02T15:04:05.999999999Z07:00"),
		Timeline:    timelineView(timeline.Entries), TimelinePage: timeline.Page,
		TimelinePages: timeline.Pages, TimelineTotal: timeline.Total,
	}
	if summary.Failure != nil {
		failure := summary.Failure
		view.Failure = &FailureView{
			Code: failure.Code, Category: failure.Category, Retryable: failure.Retryable,
			GraphID: failure.GraphID, NodeID: failure.NodeID, Attempt: failure.Attempt,
		}
	}
	return view
}

func timelinePage(entries []run.JournalEntry, page, pageSize int) ([]run.JournalEntry, int, int) {
	if pageSize <= 0 {
		pageSize = 200
	}
	if pageSize > 500 {
		pageSize = 500
	}
	pages := (len(entries) + pageSize - 1) / pageSize
	if pages == 0 {
		pages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > pages {
		page = pages
	}
	end := len(entries) - (page-1)*pageSize
	if end < 0 {
		end = 0
	}
	start := end - pageSize
	if start < 0 {
		start = 0
	}
	return entries[start:end], page, pages
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
