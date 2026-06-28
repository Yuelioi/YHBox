# Target / Controller Phase 23 Notes

SUMMARY: Phase 23 wires the default runtime controller factory into GUI and MCP execution
READ WHEN: Debugging runtime controller factory injection or adding ADB/CDP discovery
RECHECK WHEN: GUI `runFunc`, MCP `runNode`, or `DefaultControllerFactory` changes

---

Phase 23 adds default factory wiring:

- `runtime.DefaultControllerFactory` returns `AndroidADBController` for `target.KindAndroidADB`.
- Browser CDP targets return an explicit `browser cdp controller client is not wired` error until CDP discovery/connection exists.
- GUI container runs set `rt.ControllerFactory = containerruntime.DefaultControllerFactory{}` in `main.go`.
- MCP micro-runs set `rt.ControllerFactory = runtime.DefaultControllerFactory{}` in `tools_exec.go`.

Verification:

- `go test ./internal/services/container/runtime -run TestDefaultControllerFactory -count=1`
- `go test ./internal/services/mcpserver -count=1`
- `go test . -count=1`
- `go test ./internal/services/container/runtime -count=1`

Still not covered:

- Android/Browser target-selection nodes.
- ADB device discovery UI.
- CDP WebSocket discovery/client lifecycle.
