// yotta-mcp — Yotta MCP (Model Context Protocol) stdio server.
//
// This is the LLM-facing entry point for node-graph authoring: an LLM client
// connects over stdio using JSON-RPC 2.0 and calls tools to inspect the node
// catalog, build/validate container graphs, and run automation scripts.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	// Anonymous imports — trigger init() node registration so all nodes are
	// available in the registry when catalog/graph tools are called.
	_ "yotta/internal/nodes/control"
	_ "yotta/internal/nodes/detect"
	_ "yotta/internal/nodes/input"
	_ "yotta/internal/nodes/io"
	_ "yotta/internal/nodes/purefunc"
	_ "yotta/internal/nodes/stopwatch"
	_ "yotta/internal/nodes/system"
	_ "yotta/internal/nodes/variable"
)

func main() {
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
			mcp.WithDescription("Validate a Yotta container graph JSON. Returns a JSON array of structured ValidationErrors (severity error|warning, code, nodeId, pin, params). Empty array = clean. Use this to repair generated containers before saving."),
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

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("server error: %v\n", err)
		log.Fatal(err)
	}
}
