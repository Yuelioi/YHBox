# Phase 23 — App Controller Factory Wiring

## Goal

Inject runtime controller factory wiring into GUI and MCP execution paths so non-Win32 active targets have a real factory hook.

## Scope

- Add a main-package runtime controller factory.
- Return `AndroidADBController` for Android ADB targets using the default adb runner.
- Return an explicit Browser CDP wiring error until CDP client discovery exists.
- Inject the factory into:
  - GUI container runs in `main.go`;
  - MCP `run_node` micro-runs.
- Add focused tests for factory behavior.

## Non-goals

- Add Android/Browser target-selection nodes.
- Add CDP WebSocket discovery.
- Add UI discovery.

## Verification

- `go test . -run TestRuntimeControllerFactory -count=1`
- `go test ./internal/services/mcpserver -count=1`
