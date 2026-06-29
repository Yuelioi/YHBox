# Index — mcp-node-exec

## State

Implemented on local `main` as part of the landed debug/target cycle. The GUI process now starts a local Streamable HTTP MCP server at `http://127.0.0.1:8765/mcp`, with authoring tools, window lookup, `run_node`, `save_container`, armed write/execution gating, busy gating, and the Settings MCP tab.

## Next

Human smoke:
- Open Settings -> MCP and confirm the URL is visible.
- Confirm the switch says it allows execution and writes, not that it starts the server.
- Connect an MCP client to `http://127.0.0.1:8765/mcp`.
- Verify `list_nodes`, `list_windows`, and `find_window` work before arming.
- Verify `run_node` and `save_container` return `NOT_ARMED` before arming.
- Arm MCP, then use `find_window` on a simple target window and run a low-risk node such as `Capture` or `ClickAt`.
- While a normal container run is active, verify `run_node` returns `BUSY`.

## Read now

- `design.md` — product goal and tool surface.
- `plan.md` — historical implementation plan. Treat checkboxes as stale; use code/tests as source of truth.
- `../../knowledge/build/build.md` — verification baseline.

## Read if

- `../../knowledge/wails/add-service.md` — if adding frontend-visible Go services.
- `../../knowledge/nodes/node-system-architecture.md` — if changing node registration or dispatch.
- `../../knowledge/nodes/held-exec-outputs.md` — if changing `run_node` output harvesting.
- `../../knowledge/nodes/ai-nodes.md` — if aligning MCP behavior with AI node semantics.

## Progress

Done:
- Migrated authoring/schema/validate/save logic into `internal/services/mcpserver`.
- Removed legacy `cmd/yotta-mcp`.
- Added `list_nodes`, `get_graph_schema`, `validate_container`, `save_container`, `list_windows`, `find_window`, and `run_node`.
- Added `EnumTopWindows`, `ContainerRunner.ExecOutputs`, `Worker.IsRunning`, and `Settings.MCP.Armed`.
- Added micro-container harness for one-node execution and held-output/image harvesting.
- Wired the MCP server in `main.go` on `127.0.0.1:8765`.
- Added Settings MCP tab with armed gating and server URL display.

Verified:
- `go test ./internal/services/mcpserver -v`
- `go test ./pkg/winutil ./internal/services/execution ./internal/services/mcpserver`
- Landing verification also covered `go test ./...`, frontend vitest, vue-tsc, i18n check, and production build.

## Open questions

- Whether the MCP server should gain a full disable switch in addition to the current armed execution/write gate.
- Whether the fixed port `8765` needs a conflict UI or configurable port.
- Whether MCP should expose Android target discovery/run paths later, or stay Win32-window oriented for V1.
