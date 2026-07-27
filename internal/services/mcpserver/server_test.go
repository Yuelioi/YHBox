package mcpserver

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/appbootstrap"
	"github.com/yottaapp/yotta/internal/appcontrol"
	appcore "github.com/yottaapp/yotta/internal/application"
	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/httpegress"
	"github.com/yottaapp/yotta/internal/noderuntime"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/scriptengine"
	"github.com/yottaapp/yotta/internal/storage"
	"github.com/yottaapp/yotta/internal/storage/catalog"
)

func TestProtocolRegistersOnlyBoundedWorkflow31Tools(t *testing.T) {
	application := testApplication(t)
	protocol, err := BuildProtocol(application)
	if err != nil {
		t.Fatal(err)
	}
	tools := protocol.ListTools()
	want := []string{
		"catalog_search", "catalog_describe", "workflow_list", "workflow_create", "workflow_inspect",
		"workflow_apply_patch", "workflow_compile", "workflow_run_preview", "workflow_explain_diagnostic",
	}
	if len(tools) != len(want) {
		t.Fatalf("tools = %#v", tools)
	}
	for _, name := range want {
		tool, ok := tools[name]
		if !ok || len(tool.Tool.RawOutputSchema) == 0 {
			t.Fatalf("tool %q missing or lacks output schema: %#v", name, tool)
		}
	}
	for _, legacy := range []string{"list_nodes", "get_graph_schema", "validate_container", "save_container", "list_windows", "find_window", "run_node"} {
		if _, exists := tools[legacy]; exists {
			t.Fatalf("legacy tool %q is still registered", legacy)
		}
	}
}

func TestStructuredProtocolCreatesPatchesAndCompilesOneDurableRevision(t *testing.T) {
	application := testApplication(t)
	protocol, err := BuildProtocol(application)
	if err != nil {
		t.Fatal(err)
	}
	protocolClient, err := client.NewInProcessClient(protocol)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = protocolClient.Close() })
	if err := protocolClient.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	initialize := mcp.InitializeRequest{}
	initialize.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initialize.Params.ClientInfo = mcp.Implementation{Name: "yotta-test", Version: "1.0"}
	if _, err := protocolClient.Initialize(context.Background(), initialize); err != nil {
		t.Fatal(err)
	}

	createdResult := callTool(t, protocolClient, "workflow_create", map[string]any{"name": "MCP typed patch"})
	var created WorkflowSummary
	decodeStructured(t, createdResult, &created)
	if created.WorkflowID == "" || created.Revision != 0 {
		t.Fatalf("created = %#v", created)
	}

	patchedResult := callTool(t, protocolClient, "workflow_apply_patch", map[string]any{
		"workflowId": created.WorkflowID, "baseRevision": 0,
		"commands": []any{
			map[string]any{"kind": "add-node", "addNode": map[string]any{
				"graphId": "main", "nodeTypeId": nodes.ConcatNodeID, "handle": "concat",
				"position": map[string]any{"x": 0, "y": 0},
			}},
			map[string]any{"kind": "bind-value", "bindValue": map[string]any{
				"graphId": "main", "nodeId": "$concat", "portId": "a", "value": "hello",
			}},
			map[string]any{"kind": "bind-value", "bindValue": map[string]any{
				"graphId": "main", "nodeId": "$concat", "portId": "b", "value": " world",
			}},
		},
	})
	var patched WorkflowApplyPatchResult
	decodeStructured(t, patchedResult, &patched)
	if patched.Workflow.Revision != 1 || len(patched.GeneratedNodes) != 1 || patched.GeneratedNodes[0].NodeID == "" {
		t.Fatalf("patched = %#v", patched)
	}

	compiledResult := callTool(t, protocolClient, "workflow_compile", map[string]any{"workflowId": created.WorkflowID})
	var compiled WorkflowCompileResult
	decodeStructured(t, compiledResult, &compiled)
	if !compiled.SourceHash.Valid() || !compiled.ProgramHash.Valid() || len(compiled.Diagnostics) != 0 {
		t.Fatalf("compiled = %#v", compiled)
	}
}

func TestCatalogAndWorkflowEnumerationAreExplicitlyBounded(t *testing.T) {
	application := testApplication(t)
	for _, name := range []string{"one", "two", "three"} {
		if _, err := application.CreateSource(context.Background(), name); err != nil {
			t.Fatal(err)
		}
	}
	workflows, err := listWorkflows(application, PageRequest{Offset: 1, Limit: 1})
	if err != nil || workflows.Offset != 1 || workflows.Total != 3 || len(workflows.Workflows) != 1 {
		t.Fatalf("workflow page = %#v, %v", workflows, err)
	}
	projection := application.AuthoringProjection()
	catalog, err := searchCatalog(projection, CatalogSearchRequest{Limit: 1})
	if err != nil || catalog.Total <= 1 || len(catalog.Items) != 1 {
		t.Fatalf("catalog page = %#v, %v", catalog, err)
	}
	if _, err := searchCatalog(projection, CatalogSearchRequest{Limit: maxInspectPage + 1}); err == nil {
		t.Fatal("catalog accepted an unbounded page")
	}
}

func callTool(t *testing.T, protocolClient *client.Client, name string, arguments map[string]any) *mcp.CallToolResult {
	t.Helper()
	request := mcp.CallToolRequest{}
	request.Params.Name = name
	request.Params.Arguments = arguments
	result, err := protocolClient.CallTool(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("tool %s failed: %#v", name, result.Content)
	}
	if result.StructuredContent == nil {
		t.Fatalf("tool %s omitted structuredContent", name)
	}
	return result
}

func decodeStructured(t *testing.T, result *mcp.CallToolResult, target any) {
	t.Helper()
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		t.Fatalf("decode structuredContent %s: %v", raw, err)
	}
}

func testApplication(t *testing.T) *appcore.Application {
	t.Helper()
	now := time.Date(2026, 7, 15, 15, 0, 0, 0, time.UTC)
	installations, err := ai.Install(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	httpInstallations, err := httpegress.Install(nil)
	if err != nil {
		t.Fatal(err)
	}
	applicationInstallations, err := appcontrol.Install(nil)
	if err != nil {
		t.Fatal(err)
	}
	automationInstallations, err := automationinstalled.Install(nil)
	if err != nil {
		t.Fatal(err)
	}
	roots, err := storage.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	foundation, err := catalog.Open(context.Background(), roots)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = foundation.Close() })
	blobStore, err := blob.Open(roots.Objects, blob.Limits{MaxBlobBytes: 1 << 20, MaxTotalBytes: 8 << 20}, foundation.Objects())
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := appbootstrap.Build(appbootstrap.Config{
		DataRoot: roots.Data, ProgramCacheRoot: filepath.Join(roots.Cache, "programs"),
		WorkflowRepository: foundation.Workflows(),
		RunRepository:      foundation.Runs(),
		BlobStore:          blobStore,
		Limits: appbootstrap.Limits{
			MaxSources: 16, MaxPrograms: 16, MaxRuns: 16,
			MaxProgramCacheBytes:    16 << 20,
			MaxResourcePayloadBytes: 2 << 20,
			BlobChunkBytes:          64 << 10, BlobQueueCapacity: 2, StreamCapacity: 4, StreamChunkBytes: 64 << 10,
		},
		AIInstallations: installations, HTTPInstallations: httpInstallations, ApplicationInstallations: applicationInstallations, AutomationInstallations: automationInstallations, ScriptRuntime: mcpScriptRuntime(t),
		LogEmitter: noderuntime.LogEmitterFunc(func(context.Context, noderuntime.LogEntry) error { return nil }),
		GrantTTL:   time.Minute, OwnerCloseTimeout: time.Second, Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Application.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := runtime.Close(ctx); err != nil {
			t.Errorf("Close = %v", err)
		}
	})
	return runtime.Application
}

func mcpScriptRuntime(t *testing.T) *scriptengine.Runtime {
	t.Helper()
	runtime, err := scriptengine.NewRuntime(scriptengine.RuntimeOptions{
		Executable:         filepath.Join(t.TempDir(), scriptengine.WorkerExecutableName),
		ProcessMemoryBytes: scriptengine.DefaultMemoryBytes, JobMemoryBytes: scriptengine.DefaultMemoryBytes,
	})
	if err != nil {
		t.Fatal(err)
	}
	return runtime
}
