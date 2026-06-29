# Phase 71 Research — Target Preview / Picker Adapters

## Context

The runtime target/controller work is already mostly in place: YHFish has `Target`,
`Controller`, `CoordinateSpace`, target-aware input/capture adapters, Android ADB
controller support, and static target capability validation. The current product
gap is editor tooling: screen pickers, color sampling, template capture, and
container backend settings still present a Windows-first model.

This phase is research only. Do not start implementation until the adapter shape
is agreed.

## Local References Read

- `flightdeck/references/INDEX.md`
- `flightdeck/references/ok-script-findings.md`
- `flightdeck/knowledge/architecture/automation-framework-survey.md`
- `flightdeck/references/ok-script/ok/device/DeviceManager.py`
- `flightdeck/references/ok-script/ok/device/capture_methods/base.py`
- `flightdeck/references/ok-script/ok/device/capture_methods/adb.py`
- `flightdeck/references/ok-script/ok/device/capture_methods/hwnd_window.py`
- `flightdeck/references/ok-script/ok/device/interaction_methods/base.py`
- `flightdeck/references/ok-script/ok/device/interaction_methods/adb.py`
- `flightdeck/references/ok-script/ok/device/interaction_methods/post_message.py`
- `flightdeck/references/ok-script/ok/device/interaction_methods/pynput.py`
- `flightdeck/references/ok-script/ok/capture/adb/minitouch.py`
- `flightdeck/references/ok-script/ok/capture/adb/nemu_ipc.py`
- `flightdeck/references/ok-script/ok/device/capture_methods/nemu_ipc.py`
- `flightdeck/references/ok-script/ok/gui/overlay/OverlayWindow.py`
- `flightdeck/references/ok-script/ok/gui/debug/OverlayWidget.py`

## Findings

### ok-script shape

ok-script does not make each operation handle Windows/ADB/browser differences.
It first selects a preferred device, then mounts one capture method and one
interaction method:

```text
preferred device -> capture_method + interaction -> task APIs
```

The important split is not "node knows platform"; it is "backend object knows
platform". Windows-only choices such as `PostMessage` and `Pynput` live in
Windows interaction implementations. ADB interaction exposes device operations
such as tap, swipe, keyevent, text, and back. ADB capture exposes screencap; MuMu
IPC is a separate optimized capture/input path.

This supports the direction already chosen in YHFish: keep graph nodes generic
where the capability exists, and let controller/profile/capability decide whether
the active target can run the action.

### MAA/Airtest direction already recorded

The existing architecture survey records the same lesson from MaaFramework and
Airtest: controller/device is a first-class object; screenshots and input methods
are enumerable; Android ADB has multiple possible implementations such as shell
commands, minitouch/maatouch, minicap, emulator private channels, and native
device APIs.

For YHFish, this does not mean importing MaaFramework now. The useful boundary is
controller/provider/capability, plus later backend method selection. First-stage
ADB shell support is acceptable, but product tooling must stop assuming HWND.

### YHFish current shape

Runtime side:

- `internal/automation/controller/interfaces.go` already has `Controller`,
  `Screenshotter`, `PointerInput`, `KeyboardInput`, and `AppLifecycle`.
- `internal/automation/controller/profiles.go` already distinguishes Win32 and
  Android ADB capabilities.
- `internal/automation/target/types.go` already defines coordinate spaces:
  normalized, screen, window-client, capture-frame, android-device,
  browser-viewport.
- Android ADB controller supports screenshot, click, move, scroll, drag, text,
  start app, and stop app. It intentionally does not support mouse-button,
  key-state/chord, or relative mouse movement.
- Runtime initialization already avoids Win32 input/capture backends when the graph
  is Android-only.

Editor/tooling side:

- `OpenScreenPicker` is explicitly documented around Win32 context: recent
  upstream `Win32WindowTarget`, Windows screen picker window, and HWND-based
  capture.
- `PixelAt` reads OS cursor position, converts screen coordinates to a Win32
  client coordinate, then captures through a Win32 capture backend.
- Point/rect/template widgets all call `backend.tools.openScreenPicker(...)`.
- Container settings expose `inputBackend=postmessage/sendinput` and
  `captureBackend=auto/gdi/wgc/mock` as generic container options, even though
  these are Win32 backend settings.

## Product Problem

The user-facing runtime story says "Target can be Windows or Android", but the
editor tooling still says "target means Windows HWND".

For Android/MuMu this creates practical issues:

- coordinate picking cannot use the current screen picker because there is no
  HWND client coordinate;
- color sampling cannot use `PixelAt` because it depends on the desktop cursor;
- template capture/recapture cannot use the current picker because it captures
  via Windows backend;
- container settings show Windows backend knobs even when an Android target is
  the only explicit target in the graph.

## Recommended Adapter Boundary

Add a small editor-facing target tooling layer, separate from runtime node
execution:

```text
TargetToolService
  ResolveEditorTarget(containerID, nodeID) -> target.Target
  PreviewFrame(target) -> image frame + coordinate space + size
  PickPoint(target, mode) -> point/rect/template/color in target coordinate space
  SamplePixel(target, point) -> RGB/HSV
```

Adapter implementations:

- `Win32TargetPreviewAdapter`
  - may keep the current overlay/screen picker behavior;
  - uses HWND, desktop cursor, client coordinate conversion, and Win32 capture
    backend;
  - owns `postmessage/sendinput` and `gdi/wgc` semantics.
- `AndroidADBTargetPreviewAdapter`
  - uses active Android target resolution and controller screenshot;
  - shows the screenshot inside an app-owned preview/picker window;
  - converts clicks in the preview image to normalized and/or android-device
    coordinates;
  - samples pixels from the screencap image;
  - later can choose ADB screencap, MuMu IPC, minicap, etc. behind the adapter.

The Android picker should not try to overlay on the emulator window as the first
implementation. It is more stable to show a fresh screencap in YHFish and pick
inside that frame.

## Container Settings Recommendation

Rename or group current backend controls as Windows-only:

- `Windows input backend`: `postmessage`, `sendinput`
- `Windows capture backend`: `auto`, `gdi`, `wgc`, `mock`

Do not expose these as Android controls. Android ADB should use its controller
backend policy internally. If/when Android backend methods become configurable,
use Android-specific labels:

- `ADB input method`: `shell`, future `minitouch/maatouch/mumu-ipc`
- `ADB capture method`: `screencap`, future `minicap/mumu-ipc`

`scaleTolerance` can remain generic because template scaling is perception-side,
not Windows-specific.

## Suggested Implementation Slices

1. Rename UI copy only: make existing container backend settings visibly
   Windows-only. No behavior change.
2. Add backend contract tests/documentation so Windows backend settings do not
   claim Android behavior.
3. Introduce `TargetToolService` with Win32 adapter wrapping current
   `OpenScreenPicker`/`PixelAt` behavior.
4. Add Android preview screenshot endpoint using the existing Android ADB
   controller screenshot path.
5. Add Android in-app image picker for point/rect/template/color using the
   preview frame.
6. Add ADB path discovery/config later. Current Android discovery assumes `adb`
   is in PATH; MuMu often ships its own adb under the emulator installation.

## Non-goals

- Do not restore the deleted BrowserTarget product entry.
- Do not import MaaFramework as a dependency in this slice.
- Do not add minitouch/maatouch/MuMu IPC before editor preview and picker are
  target-aware.
- Do not make Android pretend to support Win32-only actions such as right mouse
  button hold, raw relative mouse movement, or key chord state unless a real
  Android implementation is designed.

