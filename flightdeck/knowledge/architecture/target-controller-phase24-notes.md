# Target Controller Upgrade — Phase 24 Notes

## Completed

- Added `node.TargetService` and `Ctx.Target()`.
- Wired runtime `RuntimeContext` active target through `NewTargetAdapter`.
- Added `AndroidTarget` node:
  - sets active `target.KindAndroidADB`
  - requires ADB serial
  - records width/height for normalized-to-device coordinate mapping
  - emits `Done.TargetID` and `Done.Kind`
- Added Chinese catalog labels for node, inputs, and outputs.

## Boundary

`AndroidTarget` does not discover devices. It is an explicit target-selection node. Discovery/UI can be layered later without changing the runtime target contract.

`WindowTarget` continues to own Win32 resolution and foreground behavior. Non-window targets should not be added to `WindowService`; use `TargetService`.

## Verification

- `go test ./internal/node ./internal/nodes/system ./internal/services/container/runtime -run "TestAndroidTarget|TestTargetAdapter|TestStubServices" -count=1`
- `go test ./internal/nodes/system ./internal/services/container/runtime -count=1`
- `go test ./internal/catalog -count=1`
- `go test ./internal/nodes/... -count=1`

## Next Risk

Input and `Capture` node routing are target-aware through the controller factory, but some vision paths still read Win32 HWND/capture cache directly. After Android target selection, template/color/QR nodes need a target-aware frame source before Android automation is complete.

