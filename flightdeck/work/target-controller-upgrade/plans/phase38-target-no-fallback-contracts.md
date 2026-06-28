# Phase 38 — Target No-Fallback Runtime Contracts

## Problem

Runtime adapters now support active targets beyond Win32 windows, but stale-window regressions are high risk:

- a graph can resolve a Win32 window first;
- a later node can switch `ActiveTarget` to Android or Browser CDP;
- input, capture, or vision must not silently reuse the previous HWND when the non-Win32 controller factory is missing or broken.

That failure mode looks like a correct target switch in graph state while screenshot picking and clicks still affect the old desktop window.

## Scope

- Add focused runtime adapter tests proving non-Win32 active targets take precedence over any previous HWND.
- Cover input click, capture screenshot, and vision `DetectColor` frame acquisition.
- Keep this as contract coverage unless tests expose a live implementation bug.

## Non-goals

- No controller implementation changes.
- No UI changes.
- No new platform support in this phase.
