// Package mcpserver projects the Application command surface into a
// bounded MCP authoring protocol. It never constructs a runtime, enumerates
// host windows, or accepts whole-document Workflow replacement.
package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/invopop/jsonschema"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	appcore "github.com/yottaapp/yotta/internal/application"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/workflow/authoring"
	"github.com/yottaapp/yotta/pkg/version"
)

type registrar struct {
	application *appcore.Application
	projection  nodeauthoring.Snapshot
	schemas     map[string]toolSchemas
}

type toolSchemas struct {
	input  json.RawMessage
	output json.RawMessage
}

// BuildProtocol creates the validated in-process MCP protocol surface. The
// returned SDK server is transport-neutral; callers may explicitly own stdio
// or a separately authenticated transport.
func BuildProtocol(application *appcore.Application) (*server.MCPServer, error) {
	registrar, err := newRegistrar(application)
	if err != nil {
		return nil, err
	}
	return registrar.protocol(), nil
}

func newRegistrar(application *appcore.Application) (*registrar, error) {
	if application == nil {
		return nil, errors.New("MCP server requires Application")
	}
	projection := application.AuthoringProjection()
	if !projection.Valid() {
		return nil, errors.New("MCP server requires trusted Authoring Projection")
	}
	schemas := make(map[string]toolSchemas, 9)
	builders := []func(map[string]toolSchemas) error{
		schemaBuilder[CatalogSearchRequest, CatalogSearchResult]("catalog_search"),
		schemaBuilder[CatalogDescribeRequest, CatalogDescribeResult]("catalog_describe"),
		schemaBuilder[PageRequest, WorkflowListResult]("workflow_list"),
		schemaBuilder[WorkflowCreateRequest, WorkflowSummary]("workflow_create"),
		schemaBuilder[WorkflowInspectRequest, WorkflowInspectResult]("workflow_inspect"),
		schemaBuilder[WorkflowApplyPatchRequest, WorkflowApplyPatchResult]("workflow_apply_patch"),
		schemaBuilder[WorkflowIDRequest, WorkflowCompileResult]("workflow_compile"),
		schemaBuilder[WorkflowIDRequest, WorkflowRunPreviewResult]("workflow_run_preview"),
		schemaBuilder[ExplainDiagnosticRequest, ExplainDiagnosticResult]("workflow_explain_diagnostic"),
	}
	for _, build := range builders {
		if err := build(schemas); err != nil {
			return nil, err
		}
	}
	patchSchema, err := authoring.GenerateSchema()
	if err != nil {
		return nil, fmt.Errorf("build MCP workflow_apply_patch input schema: %w", err)
	}
	patchTool := schemas["workflow_apply_patch"]
	patchTool.input = patchSchema
	schemas["workflow_apply_patch"] = patchTool
	return &registrar{application: application, projection: projection, schemas: schemas}, nil
}

// protocol constructs an in-process MCP protocol server with output schema
// validation. Transport ownership is intentionally outside this package; the
// desktop application does not open an MCP listener by default.
func (s *registrar) protocol() *server.MCPServer {
	protocol := server.NewMCPServer(
		"Yotta Workflow Authoring", version.Version,
		server.WithToolCapabilities(false),
		server.WithOutputSchemaValidation(),
	)
	s.register(protocol)
	return protocol
}

func (s *registrar) register(protocol *server.MCPServer) {
	protocol.AddTool(s.tool(
		"catalog_search", "Search the admitted Yotta node catalog without loading the full projection."),
		mcp.NewStructuredToolHandler(func(_ context.Context, _ mcp.CallToolRequest, request CatalogSearchRequest) (CatalogSearchResult, error) {
			return searchCatalog(s.projection, request)
		}),
	)
	protocol.AddTool(s.tool(
		"catalog_describe", "Describe one exact Node Contract with typed channel-separated ports and authoring constraints."),
		mcp.NewStructuredToolHandler(func(_ context.Context, _ mcp.CallToolRequest, request CatalogDescribeRequest) (CatalogDescribeResult, error) {
			return describeCatalog(s.projection, request)
		}),
	)
	protocol.AddTool(s.tool(
		"workflow_list", "List durable Workflow Sources as bounded metadata."),
		mcp.NewStructuredToolHandler(func(_ context.Context, _ mcp.CallToolRequest, request PageRequest) (WorkflowListResult, error) {
			return listWorkflows(s.application, request)
		}),
	)
	protocol.AddTool(s.mutationTool("workflow_create", "Create the host-owned empty Workflow root."),
		mcp.NewStructuredToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, request WorkflowCreateRequest) (WorkflowSummary, error) {
			return createWorkflow(ctx, s.application, request)
		}),
	)
	protocol.AddTool(s.tool(
		"workflow_inspect", "Inspect one graph page from a durable Workflow revision."),
		mcp.NewStructuredToolHandler(func(_ context.Context, _ mcp.CallToolRequest, request WorkflowInspectRequest) (WorkflowInspectResult, error) {
			return inspectWorkflow(s.application, request)
		}),
	)
	protocol.AddTool(s.mutationTool("workflow_apply_patch", "Atomically apply typed domain commands with exact baseRevision CAS; IDs and defaults are host-owned."),
		mcp.NewStructuredToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, request WorkflowApplyPatchRequest) (WorkflowApplyPatchResult, error) {
			return applyWorkflowPatch(ctx, s.application, request)
		}),
	)
	protocol.AddTool(s.tool(
		"workflow_compile", "Strictly compile the current durable Workflow revision and return stable diagnostics."),
		mcp.NewStructuredToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, request WorkflowIDRequest) (WorkflowCompileResult, error) {
			return compileWorkflow(ctx, s.application, request)
		}),
	)
	protocol.AddTool(s.tool(
		"workflow_run_preview", "Compile the durable revision and show the exact capability plan without admission or effects."),
		mcp.NewStructuredToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, request WorkflowIDRequest) (WorkflowRunPreviewResult, error) {
			return previewWorkflowRun(ctx, s.application, request)
		}),
	)
	protocol.AddTool(s.tool(
		"workflow_explain_diagnostic", "Explain one stable compiler diagnostic code and bounded repair choices."),
		mcp.NewStructuredToolHandler(func(_ context.Context, _ mcp.CallToolRequest, request ExplainDiagnosticRequest) (ExplainDiagnosticResult, error) {
			return explainDiagnostic(request)
		}),
	)
}

func (s *registrar) tool(name, description string) mcp.Tool {
	schemas := s.schemas[name]
	return mcp.NewTool(name,
		mcp.WithDescription(description), mcp.WithRawInputSchema(schemas.input), mcp.WithRawOutputSchema(schemas.output),
		mcp.WithReadOnlyHintAnnotation(true), mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true), mcp.WithOpenWorldHintAnnotation(false),
	)
}

func (s *registrar) mutationTool(name, description string) mcp.Tool {
	schemas := s.schemas[name]
	return mcp.NewTool(name,
		mcp.WithDescription(description), mcp.WithRawInputSchema(schemas.input), mcp.WithRawOutputSchema(schemas.output),
		mcp.WithReadOnlyHintAnnotation(false), mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false), mcp.WithOpenWorldHintAnnotation(false),
	)
}

func schemaBuilder[Input, Output any](name string) func(map[string]toolSchemas) error {
	return func(destination map[string]toolSchemas) error {
		input, err := reflectSchema[Input]()
		if err != nil {
			return fmt.Errorf("build MCP %s input schema: %w", name, err)
		}
		output, err := reflectSchema[Output]()
		if err != nil {
			return fmt.Errorf("build MCP %s output schema: %w", name, err)
		}
		destination[name] = toolSchemas{input: input, output: output}
		return nil
	}
}

func reflectSchema[Value any]() (json.RawMessage, error) {
	reflector := jsonschema.Reflector{Anonymous: true, ExpandedStruct: true, DoNotReference: false}
	contract := reflector.Reflect(new(Value))
	raw, err := json.Marshal(contract)
	if err != nil {
		return nil, err
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	if root["type"] != "object" {
		return nil, errors.New("MCP tool schema root must be an object")
	}
	root["additionalProperties"] = false
	return json.Marshal(root)
}
