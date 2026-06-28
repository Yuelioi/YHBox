# Phase 24 — Android ADB Target Node

## Goal

Allow a graph to switch the runtime active target to an Android ADB device without abusing `WindowTarget`.

## Scope

- Add a `node.TargetService` exposed through `node.Ctx`.
- Wire runtime `RuntimeContext.SetActiveTarget` / `ActiveTarget` through that service.
- Add a minimal `AndroidTarget` system node:
  - inputs: `Serial`, `Width`, `Height`, optional `Name`
  - output: `Done.TargetID`
  - behavior: validate and set active `target.KindAndroidADB`
- Add catalog translations for the new node and pins.
- Cover node/service/runtime adapter behavior with focused tests.

## Non-goals

- No ADB device auto-discovery UI in this phase.
- No app start/stop node in this phase.
- No Browser CDP target discovery in this phase.

## Verification

- `go test ./internal/node ./internal/nodes/system ./internal/services/container/runtime -run "TestAndroidTarget|TestTargetService|TestRuntimeTarget|TestStubServices" -count=1`
- `go test ./internal/nodes/system ./internal/services/container/runtime -count=1`

