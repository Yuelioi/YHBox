# Target / Controller Phase 12 Notes

SUMMARY: Phase 12 routes Capture service full-frame acquisition through Win32Controller.Screenshot
READ WHEN: Debugging Capture node traces / changing capture routing / planning screenshot target support
RECHECK WHEN: `captureAdapter`, `runtimeWin32Capture`, `Win32Controller.Screenshot`, or Capture node behavior changes

---

Phase 12 migrates Capture service acquisition:

- `runtimeWin32Capture` adapts `pkg/capture.IBackend` to `controller.Win32Capture`.
- `captureAdapter.Capture` and `CaptureROI` acquire the full frame through `Win32Controller.Screenshot`.
- Existing runtime PNG encoding and ROI crop behavior remain in `captureAdapter`.
- Per-node `ServiceBundle` copies now replace both `Input` and `Capture` with source-aware adapters.

Operational effect:

- `Capture` node emits a `screenshot` trace when dispatched through the framework.
- The trace includes Phase 8 source metadata.
- Direct `NewCaptureAdapter(rt)` calls keep empty source metadata.

Still not covered:

- Vision adapter capture paths and cached frame reads.
- Controller-native ROI screenshot/crop.
- Trace viewer and persistence.
- Browser/Android screenshot implementations.
