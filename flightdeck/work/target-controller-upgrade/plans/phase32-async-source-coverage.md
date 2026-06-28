# Phase 32 — Async source coverage

## Goal

Prevent async dropdown fields from shipping with an unregistered backend source. `PinInput` calls `NodeService.AsyncOptions` for every `async-dropdown`; if the source is missing, the user sees a runtime dropdown error.

## Scope

- Register the existing `clipIDs` and `subgraphIDs` sources.
- Keep Android ADB and Browser CDP source registration unchanged.
- Add a coverage test that compares all async sources declared by node specs with the sources registered by the app-level composition helpers.
- Expose registered source names from `NodeService` for diagnostics/tests.

## Verification

- `go test ./internal/services/nodeoptions ./internal/node ./internal/nodes/io ./internal/nodes/system -count=1`
- Relevant broader Go tests after wiring `main.go`.
