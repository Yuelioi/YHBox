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

	"github.com/yottaapp/yotta/internal/apperr"
	"github.com/yottaapp/yotta/internal/workflow/authoring"
	"github.com/yottaapp/yotta/internal/workflowbundle"
)

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
	return SourcePage{
		Items: append([]SourceView(nil), filtered[start:end]...),
		Total: total, Page: query.Page, PageSize: query.PageSize,
		Categories: categories, Tags: tags,
	}, nil
}

func (s *Service) BatchUpdateSourceMetadata(requests []BatchUpdateSourceMetadataRequest) []BatchUpdateSourceMetadataResult {
	results := make([]BatchUpdateSourceMetadataResult, 0, len(requests))
	seen := make(map[string]struct{}, len(requests))
	for _, request := range requests {
		result := BatchUpdateSourceMetadataResult{WorkflowID: request.WorkflowID}
		if _, duplicate := seen[request.WorkflowID]; duplicate {
			result.Problem = workflowProblem("workflow.batch.duplicate", map[string]any{"operation": "metadata"})
			results = append(results, result)
			continue
		}
		seen[request.WorkflowID] = struct{}{}
		snapshot, err := s.application.GetSource(request.WorkflowID)
		var current SourceView
		if err == nil {
			current, err = sourceView(snapshot, false)
		}
		if err == nil {
			_, err = s.application.ApplyPatch(context.Background(), authoring.PatchRequest{
				WorkflowID: request.WorkflowID, BaseRevision: request.BaseRevision,
				Commands: []authoring.Command{{
					Kind: authoring.CommandUpdateWorkflowMetadata,
					UpdateWorkflowMetadata: &authoring.UpdateWorkflowMetadataCommand{
						Name: current.Name, Description: current.Description,
						Category: request.Category, Tags: request.Tags,
					},
				}},
			})
		}
		if err != nil {
			result.Problem = workflowProblemFrom(err)
		} else {
			result.Updated = true
		}
		results = append(results, result)
	}
	return results
}

func (s *Service) ExportSourceBundles(workflowIDs []string, directory string) []BundleExportResult {
	results := make([]BundleExportResult, 0, len(workflowIDs))
	seen := make(map[string]struct{}, len(workflowIDs))
	for _, workflowID := range workflowIDs {
		result := BundleExportResult{WorkflowID: workflowID}
		if s.bundles == nil {
			result.Problem = workflowProblem("workflow.bundle.unavailable", nil)
		} else if strings.TrimSpace(directory) == "" {
			result.Problem = workflowProblem("workflow.bundle.directory_required", nil)
		} else if _, duplicate := seen[workflowID]; duplicate {
			result.Problem = workflowProblem("workflow.batch.duplicate", map[string]any{"operation": "export"})
		} else {
			seen[workflowID] = struct{}{}
			destination := filepath.Join(directory, workflowID+workflowbundle.Extension)
			if _, err := os.Lstat(destination); err == nil {
				result.Problem = workflowProblem("workflow.bundle.destination_exists", map[string]any{"workflowId": workflowID})
			} else if !errors.Is(err, os.ErrNotExist) {
				result.Problem = workflowProblemFrom(err)
			} else if exported, err := s.bundles.Export(context.Background(), workflowID, destination); err != nil {
				result.Problem = workflowProblemFrom(err)
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
		result = append(result, DeleteSourcePreview{
			WorkflowID: workflowID, Name: view.Name,
			References: s.sourceReferences(workflowID),
		})
	}
	return result, nil
}

func (s *Service) DeleteSources(requests []DeleteSourceRequest) []DeleteSourceResult {
	results := make([]DeleteSourceResult, 0, len(requests))
	seen := make(map[string]struct{}, len(requests))
	for _, request := range requests {
		result := DeleteSourceResult{WorkflowID: request.WorkflowID}
		if _, duplicate := seen[request.WorkflowID]; duplicate {
			result.Problem = workflowProblem("workflow.batch.duplicate", map[string]any{"operation": "delete"})
			results = append(results, result)
			continue
		}
		seen[request.WorkflowID] = struct{}{}
		result.References = s.sourceReferences(request.WorkflowID)
		if len(result.References) != 0 {
			result.Problem = workflowProblem("workflow.source.referenced", map[string]any{"references": len(result.References)})
			results = append(results, result)
			continue
		}
		if err := s.application.DeleteSource(context.Background(), request.WorkflowID, request.Revision, request.SourceHash); err != nil {
			result.Problem = workflowProblemFrom(err)
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
	return since.IsZero() || !sourceTimestamp(value).Before(since)
}

func sortSourceViewsByTime(views []SourceView, value func(SourceView) string) {
	sort.SliceStable(views, func(i, j int) bool {
		left, right := sourceTimestamp(value(views[i])), sourceTimestamp(value(views[j]))
		if left.Equal(right) {
			return strings.ToLower(views[i].Name) < strings.ToLower(views[j].Name)
		}
		return left.After(right)
	})
}

func sourceTimestamp(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, value)
	return parsed
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

func workflowProblem(id string, params map[string]any) *apperr.Envelope {
	envelope := apperr.From(apperr.New(id, params))
	return &envelope
}

func workflowProblemFrom(err error) *apperr.Envelope {
	envelope := apperr.From(err)
	return &envelope
}
