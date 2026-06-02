// yotta-mcp — Yotta MCP (Model Context Protocol) stdio server.
//
// This is the LLM-facing entry point for node-graph authoring: an LLM client
// connects over stdio using JSON-RPC 2.0 and calls tools to inspect the node
// catalog, build/validate container graphs, and run automation scripts.
//
// Task 4 scaffolds the minimal skeleton with a single `ping` tool.
// More tools (catalog_list, graph_validate, graph_run, …) come in later tasks.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	// Anonymous imports — trigger init() node registration so all nodes are
	// available in the registry when catalog/graph tools are added later.
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

	pingTool := mcp.NewTool("ping",
		mcp.WithDescription("Liveness check — returns \"pong\"."),
	)
	s.AddTool(pingTool, func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("pong"), nil
	})

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("server error: %v\n", err)
		log.Fatal(err)
	}
}
