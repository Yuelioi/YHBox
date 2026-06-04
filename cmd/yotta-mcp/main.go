// yotta-mcp — Yotta MCP (Model Context Protocol) stdio server.
//
// This is the LLM-facing entry point for node-graph authoring: an LLM client
// connects over stdio using JSON-RPC 2.0 and calls tools to inspect the node
// catalog, build/validate container graphs, and run automation scripts.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"yotta/internal/services/container"

	// Anonymous imports — trigger init() node registration so all nodes are
	// available in the registry when catalog/graph tools are called.
	_ "yotta/internal/nodes/control"
	_ "yotta/internal/nodes/detect"
	_ "yotta/internal/nodes/event"     // EventTick (listener-driven 定时触发)
	_ "yotta/internal/nodes/input"
	_ "yotta/internal/nodes/io"
	_ "yotta/internal/nodes/purefunc"
	_ "yotta/internal/nodes/stopwatch"
	_ "yotta/internal/nodes/system"
	_ "yotta/internal/nodes/variable"
)

func main() {
	// 数据根：镜像主 main.go 的解析逻辑。
	// 优先读 YOTTA_DATA_DIR 环境变量（主 app 启动时会 Setenv，MCP 独立跑时用户可手动设）；
	// 为空则回落到 <exeDir>/data，exeDir 读失败再回落 "data"。
	dataDir := os.Getenv("YOTTA_DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
		if exe, err := os.Executable(); err == nil {
			dataDir = filepath.Join(filepath.Dir(exe), "data")
		}
	}
	st, err := container.NewStore(filepath.Join(dataDir, "containers"))
	if err != nil {
		log.Fatalf("container store init: %v", err)
	}

	s := server.NewMCPServer("yotta-mcp", "0.1.0")

	s.AddTool(
		mcp.NewTool("list_nodes",
			mcp.WithDescription("List all available Yotta node kinds with their pins (inputs/outputs), types, required flags, defaults, category, and capability flags. The building blocks for authoring a container graph.")),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(string(listNodesJSON())), nil
		},
	)

	s.AddTool(
		mcp.NewTool("get_graph_schema",
			mcp.WithDescription("Return the Yotta container-graph JSON schema conventions and validated example containers. Read this before generating a container.")),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(string(graphSchemaJSON())), nil
		},
	)

	s.AddTool(
		mcp.NewTool("validate_container",
			mcp.WithDescription("Validate a Yotta container graph JSON. Returns a JSON array of structured ValidationErrors (severity error|warning, code, nodeId, graphPath, params — pin/kind/etc. live inside params). Empty array = clean. Use this to repair generated containers before saving."),
			mcp.WithString("container",
				mcp.Required(),
				mcp.Description("The container graph JSON string to validate."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			raw, err := req.RequireString("container")
			if err != nil || raw == "" {
				return mcp.NewToolResultError("missing 'container' argument"), nil
			}
			out, _ := validateContainerJSON([]byte(raw))
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	s.AddTool(
		mcp.NewTool("save_container",
			mcp.WithDescription("Validate and persist a Yotta container graph. Rejects (returns ValidationErrors) if there are error-level issues; warnings do not block. The server assigns the container id; the input id is ignored. Returns {id, path, warnings}."),
			mcp.WithString("container", mcp.Required(), mcp.Description("The container graph JSON.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			raw, err := req.RequireString("container")
			if err != nil {
				return mcp.NewToolResultError("missing 'container' argument"), nil
			}
			res, saveErrs := saveContainer(st, []byte(raw))
			if saveErrs != nil {
				return mcp.NewToolResultError(string(saveErrs)), nil
			}
			b, _ := json.MarshalIndent(res, "", "  ")
			return mcp.NewToolResultText(string(b)), nil
		},
	)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("server error: %v\n", err)
		log.Fatal(err)
	}
}
