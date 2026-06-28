# Phase 27 — Frontend Async Dropdown

## Goal

Render backend `WidgetSpec{Kind:"async-dropdown"}` as a usable inspector control that loads options through `NodeService.AsyncOptions`.

## Scope

- Preserve manual text entry fallback.
- Load options lazily when the control mounts or source changes.
- Show loading/error states without blocking the inspector.
- Use current node inputs as async params where available.
- Verify `AndroidTarget.Serial` can surface `androidADBDevices` options.

## Non-goals

- No auto-filling AndroidTarget width/height from selected option.
- No disabled/offline device option states.
- No browser/CDP target picker.

## Verification

- `pnpm vue-tsc --noEmit`
- focused frontend tests if an existing test harness covers `PinInput`; otherwise run related store/component tests that compile the inspector path.

