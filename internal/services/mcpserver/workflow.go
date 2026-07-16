package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"strings"

	appcore "github.com/yottaapp/yotta/internal/application"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/workflow/authoring"
	"github.com/yottaapp/yotta/internal/workflow/schema"
	"github.com/yottaapp/yotta/internal/workflowstore"
)

const maxInspectPage = 100

type PageRequest struct {
	Offset int `json:"offset,omitempty" jsonschema:"minimum=0"`
	Limit  int `json:"limit,omitempty" jsonschema:"minimum=1,maximum=100"`
}

type WorkflowIDRequest struct {
	WorkflowID string `json:"workflowId" jsonschema:"required,description=Workflow identifier"`
}

type WorkflowCreateRequest struct {
	Name string `json:"name" jsonschema:"required,description=Human readable workflow name"`
}

type WorkflowSummary struct {
	WorkflowID    string          `json:"workflowId"`
	Name          string          `json:"name"`
	Revision      int64           `json:"revision"`
	SourceHash    artifact.Digest `json:"sourceHash"`
	EntryGraph    string          `json:"entryGraph"`
	GraphCount    int             `json:"graphCount"`
	VariableCount int             `json:"variableCount"`
	SecretCount   int             `json:"secretCount"`
}

type WorkflowListResult struct {
	Offset    int               `json:"offset"`
	Total     int               `json:"total"`
	Workflows []WorkflowSummary `json:"workflows"`
}

type WorkflowInspectRequest struct {
	WorkflowID string `json:"workflowId" jsonschema:"required,description=Workflow identifier"`
	GraphID    string `json:"graphId,omitempty" jsonschema:"description=Graph identifier defaults to entry graph"`
	NodeOffset int    `json:"nodeOffset,omitempty" jsonschema:"minimum=0"`
	EdgeOffset int    `json:"edgeOffset,omitempty" jsonschema:"minimum=0"`
	Limit      int    `json:"limit,omitempty" jsonschema:"minimum=1,maximum=100"`
}

type GraphPage struct {
	GraphID    string        `json:"graphId"`
	Kind       string        `json:"kind"`
	NodeOffset int           `json:"nodeOffset"`
	EdgeOffset int           `json:"edgeOffset"`
	NodeTotal  int           `json:"nodeTotal"`
	EdgeTotal  int           `json:"edgeTotal"`
	Nodes      []schema.Node `json:"nodes"`
	Edges      []schema.Edge `json:"edges"`
}

type WorkflowInspectResult struct {
	Workflow WorkflowSummary `json:"workflow"`
	Graph    GraphPage       `json:"graph"`
}

type WorkflowApplyPatchRequest = authoring.PatchRequest

type WorkflowApplyPatchResult struct {
	Workflow       WorkflowSummary           `json:"workflow"`
	GeneratedNodes []authoring.GeneratedNode `json:"generatedNodes"`
}

type WorkflowCompileResult struct {
	WorkflowID  string              `json:"workflowId"`
	SourceHash  artifact.Digest     `json:"sourceHash,omitempty"`
	ProgramHash artifact.Digest     `json:"programHash,omitempty"`
	Diagnostics []schema.Diagnostic `json:"diagnostics"`
}

type WorkflowRunPreviewResult struct {
	WorkflowID     string                 `json:"workflowId"`
	SourceHash     artifact.Digest        `json:"sourceHash,omitempty"`
	ProgramHash    artifact.Digest        `json:"programHash,omitempty"`
	Diagnostics    []schema.Diagnostic    `json:"diagnostics"`
	CapabilityPlan []capability.PlanEntry `json:"capabilityPlan"`
}

type ExplainDiagnosticRequest struct {
	Code string `json:"code" jsonschema:"required,description=Stable compiler diagnostic code"`
}

type ExplainDiagnosticResult struct {
	Code        string   `json:"code"`
	Explanation string   `json:"explanation"`
	Repairs     []string `json:"repairs"`
}

func listWorkflows(application *appcore.Application, request PageRequest) (WorkflowListResult, error) {
	offset, limit, err := boundedPage(request.Offset, request.Limit)
	if err != nil {
		return WorkflowListResult{}, err
	}
	snapshots := application.ListSources()
	result := WorkflowListResult{
		Offset: offset, Total: len(snapshots),
		Workflows: make([]WorkflowSummary, 0, min(limit, len(snapshots))),
	}
	snapshots = page(snapshots, offset, limit)
	for _, snapshot := range snapshots {
		summary, err := summarizeSource(snapshot)
		if err != nil {
			return WorkflowListResult{}, err
		}
		result.Workflows = append(result.Workflows, summary)
	}
	return result, nil
}

func createWorkflow(ctx context.Context, application *appcore.Application, request WorkflowCreateRequest) (WorkflowSummary, error) {
	snapshot, err := application.CreateSource(ctx, request.Name)
	if err != nil {
		return WorkflowSummary{}, err
	}
	return summarizeSource(snapshot)
}

func inspectWorkflow(application *appcore.Application, request WorkflowInspectRequest) (WorkflowInspectResult, error) {
	if request.NodeOffset < 0 || request.EdgeOffset < 0 {
		return WorkflowInspectResult{}, errors.New("offsets must be non-negative")
	}
	limit := request.Limit
	if limit == 0 {
		limit = 50
	}
	if limit < 1 || limit > maxInspectPage {
		return WorkflowInspectResult{}, fmt.Errorf("limit must be between 1 and %d", maxInspectPage)
	}
	snapshot, err := application.GetSource(request.WorkflowID)
	if err != nil {
		return WorkflowInspectResult{}, err
	}
	source, diagnostics := schema.ParseSource(snapshot.Artifact())
	if len(diagnostics) != 0 {
		return WorkflowInspectResult{}, errors.New("stored Workflow Source failed strict reopen")
	}
	graphID := request.GraphID
	if graphID == "" {
		graphID = source.EntryGraph
	}
	var graph *schema.Graph
	for index := range source.Graphs {
		if source.Graphs[index].ID == graphID {
			graph = &source.Graphs[index]
			break
		}
	}
	if graph == nil {
		return WorkflowInspectResult{}, fmt.Errorf("UNKNOWN_GRAPH: %s", graphID)
	}
	summary, err := summarizeSource(snapshot)
	if err != nil {
		return WorkflowInspectResult{}, err
	}
	return WorkflowInspectResult{Workflow: summary, Graph: GraphPage{
		GraphID: graph.ID, Kind: string(graph.Kind), NodeOffset: request.NodeOffset, EdgeOffset: request.EdgeOffset,
		NodeTotal: len(graph.Nodes), EdgeTotal: len(graph.Edges),
		Nodes: page(graph.Nodes, request.NodeOffset, limit), Edges: page(graph.Edges, request.EdgeOffset, limit),
	}}, nil
}

func applyWorkflowPatch(ctx context.Context, application *appcore.Application, request WorkflowApplyPatchRequest) (WorkflowApplyPatchResult, error) {
	result, err := application.ApplyPatch(ctx, request)
	if err != nil {
		return WorkflowApplyPatchResult{}, err
	}
	summary, err := summarizeSource(result.Source)
	if err != nil {
		return WorkflowApplyPatchResult{}, err
	}
	return WorkflowApplyPatchResult{
		Workflow: summary, GeneratedNodes: append([]authoring.GeneratedNode{}, result.GeneratedNodes...),
	}, nil
}

func compileWorkflow(ctx context.Context, application *appcore.Application, request WorkflowIDRequest) (WorkflowCompileResult, error) {
	compiled, err := application.CompileSource(ctx, request.WorkflowID)
	result := WorkflowCompileResult{
		WorkflowID: request.WorkflowID, SourceHash: compiled.SourceHash,
		Diagnostics: append([]schema.Diagnostic{}, compiled.Diagnostics...),
	}
	if program, ok := compiled.Program(); ok {
		result.ProgramHash = program.Hash()
	}
	return result, err
}

func previewWorkflowRun(ctx context.Context, application *appcore.Application, request WorkflowIDRequest) (WorkflowRunPreviewResult, error) {
	preview, err := application.PreviewRun(ctx, request.WorkflowID)
	return WorkflowRunPreviewResult{
		WorkflowID: request.WorkflowID, SourceHash: preview.SourceHash, ProgramHash: preview.ProgramHash,
		Diagnostics:    append([]schema.Diagnostic{}, preview.Diagnostics...),
		CapabilityPlan: append([]capability.PlanEntry{}, preview.CapabilityPlan...),
	}, err
}

func explainDiagnostic(request ExplainDiagnosticRequest) (ExplainDiagnosticResult, error) {
	code := strings.TrimSpace(request.Code)
	explanations := map[string]ExplainDiagnosticResult{
		"UNKNOWN_NODE_TYPE":      {Explanation: "The source pins a node type that is absent from the admitted Catalog.", Repairs: []string{"Use catalog_search and catalog_describe, then add a supported node."}},
		"NODE_CONTRACT_MISMATCH": {Explanation: "The node semantic digest does not match the admitted Node Contract.", Repairs: []string{"Remove and re-add the node through workflow_apply_patch."}},
		"UNKNOWN_PORT":           {Explanation: "An edge or binding names a port that the pinned Node Contract does not declare.", Repairs: []string{"Inspect the node contract and patch the endpoint or binding."}},
		"EDGE_CHANNEL_MISMATCH":  {Explanation: "The edge channel does not match the source output or the target instruction's accepted input channel.", Repairs: []string{"Inspect the exact node instruction and connect data, normal execution, or an explicit error route to a compatible input."}},
		"REGION_SIGNAL_SCOPE":    {Explanation: "A break, continue, or retry signal originates outside the activation body owned by that region node.", Repairs: []string{"Route the signal from a node reachable through the region's body output, using the instruction-declared channel."}},
		"TYPE_MISMATCH":          {Explanation: "The connected data types cannot be unified exactly.", Repairs: []string{"Insert an explicit conversion node or choose a compatible port."}},
		"MISSING_INPUT_BINDING":  {Explanation: "A required data input has neither a literal/default binding nor one incoming data edge.", Repairs: []string{"Bind a value/default or connect exactly one data source."}},
		"INVALID_CONFIG":         {Explanation: "Node config does not satisfy the exact Node Contract schema.", Repairs: []string{"Use catalog_describe to inspect fields and constraints, then set the field explicitly."}},
		"NO_EXECUTION_ROOT":      {Explanation: "The graph has no event root and no pull-data sink from which execution can begin.", Repairs: []string{"Add an event root or a meaningful terminal data sink."}},
		"DATA_CYCLE":             {Explanation: "Pull-data dependencies contain a cycle.", Repairs: []string{"Break the data cycle; state and control flow must be explicit nodes."}},
	}
	result, ok := explanations[code]
	if !ok {
		return ExplainDiagnosticResult{}, fmt.Errorf("UNKNOWN_DIAGNOSTIC: %s", code)
	}
	result.Code = code
	return result, nil
}

func summarizeSource(snapshot workflowstore.SourceSnapshot) (WorkflowSummary, error) {
	source, diagnostics := schema.ParseSource(snapshot.Artifact())
	if len(diagnostics) != 0 {
		return WorkflowSummary{}, errors.New("stored Workflow Source failed strict reopen")
	}
	return WorkflowSummary{
		WorkflowID: source.Workflow.ID, Name: source.Workflow.Name, Revision: source.Revision,
		SourceHash: snapshot.Hash(), EntryGraph: source.EntryGraph, GraphCount: len(source.Graphs),
		VariableCount: len(source.Variables), SecretCount: len(source.SecretRefs),
	}, nil
}

func page[T any](values []T, offset, limit int) []T {
	if offset >= len(values) {
		return []T{}
	}
	end := offset + limit
	if end > len(values) {
		end = len(values)
	}
	return append([]T{}, values[offset:end]...)
}

func boundedPage(offset, limit int) (int, int, error) {
	if offset < 0 {
		return 0, 0, errors.New("offset must be non-negative")
	}
	if limit == 0 {
		limit = 50
	}
	if limit < 1 || limit > maxInspectPage {
		return 0, 0, fmt.Errorf("limit must be between 1 and %d", maxInspectPage)
	}
	return offset, limit, nil
}
