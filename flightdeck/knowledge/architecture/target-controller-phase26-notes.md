# Target Controller Upgrade — Phase 26 Notes

## Completed

- Added `internal/services/androidadb`.
- Added `Service.ListDevices(ctx)`:
  - parses `adb devices -l`
  - queries `adb -s <serial> shell wm size` for online devices
  - keeps devices when resolution query fails
- Added parsers for device list and `wm size` output.
- Added `androidADBDevices` NodeService async source.
- Registered the async source in `main.go`.
- Changed `AndroidTarget.Serial` widget metadata to `async-dropdown` using `androidADBDevices`.

## Boundary

The async source only returns devices with `State == "device"`. Unauthorized/offline devices are parsed internally but not returned to the dropdown because `node.EnumOption` has no disabled/status field yet.

Frontend generic `async-dropdown` is not implemented in this phase. The backend source is available through `NodeService.AsyncOptions`, but the inspector may still fall back to text input until Phase 27.

## Verification

- `go test ./internal/services/androidadb ./internal/node ./internal/nodes/system . -count=1`

## Next Risk

Implement a generic frontend async-dropdown renderer that calls `NodeService.AsyncOptions`, preserves manual entry fallback, and avoids blocking the inspector on slow device discovery.

