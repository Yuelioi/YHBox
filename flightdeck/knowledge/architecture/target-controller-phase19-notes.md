# Target / Controller Phase 19 Notes

SUMMARY: Phase 19 adds a testable Android ADB controller skeleton
READ WHEN: Extending Android automation, emulator support, or Maa-style backend integration
RECHECK WHEN: ADB command mapping, Android target resolution, or controller runtime routing changes

---

Phase 19 adds `AndroidADBController` under `internal/automation/controller`:

- Accepts only `target.KindAndroidADB`.
- Uses injected `ADBRunner` so tests do not require a real device.
- Defaults to running `adb -s <serial> ...` when no runner is injected.
- Implements:
  - screenshot: `exec-out screencap -p`;
  - click: `shell input tap x y`;
  - move: zero-duration `shell input swipe x y x y 0`;
  - drag: `shell input swipe x1 y1 x2 y2 duration`;
  - scroll: `shell input swipe` from a point with notch-derived delta;
  - text: `shell input text`, with space / percent escaping;
  - start app: `shell monkey -p <package> -c android.intent.category.LAUNCHER 1`;
  - stop app: `shell am force-stop <package>`.
- Normalized points require target resolution and clamp to device bounds.
- Actions emit existing `automation/trace.ActionRecord` records.

Verification:

- `go test ./internal/automation/controller -count=1`
- `go test ./internal/automation/... -count=1`

Still not covered:

- Runtime routing to Android targets.
- Device discovery / emulator selection UI.
- MaaFramework adapter layer.
- Advanced Android key event mapping.
