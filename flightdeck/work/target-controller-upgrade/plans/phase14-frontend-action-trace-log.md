# Phase 14 Frontend Action Trace Log Consumer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Consume `container:action-trace` events in the frontend and expose them through the existing log store.

**Architecture:** Extend `useLogStore` with a bounded `actionTraces` cache and an `appendActionTrace` action that also appends a compact `action` log line. Wire the Wails event in `frontend/src/lib/events.ts`. Avoid a new UI panel in this phase; existing `LogPanel` immediately shows action lines, and future trace UI can consume the structured cache.

**Tech Stack:** Vue, Pinia, Vitest, Wails Events.

---

## Scope

In scope:

- Add frontend `ActionTraceEntry` type.
- Add `actionTraces` bounded cache to `useLogStore`.
- Add `appendActionTrace` formatter.
- Wire raw `container:action-trace` event to the log store.
- Add store tests for structured cache, compact log line, and clear behavior.

Out of scope:

- Dedicated trace viewer panel.
- Timeline visualization.
- Persistence.
- Payload redaction UI.

## Design Notes

- Action trace lines use log level `action` and source `CTR`.
- The structured cache should be capped with the same ring capacity as log lines.
- `clear()` clears both log lines and action traces.

## Tasks

1. Commit this plan and update `flightdeck/work/target-controller-upgrade/index.md`.
2. Add failing log store test for `appendActionTrace`.
3. Implement store action/cache and log formatting.
4. Wire `container:action-trace` in `lib/events.ts`.
5. Run focused frontend tests/typecheck.
6. Add Phase 14 notes, update index, and commit docs.

## Verification

```powershell
pnpm --dir frontend vitest run src/stores/__tests__/log.spec.ts src/stores/log.spec.ts
pnpm --dir frontend vue-tsc --noEmit
```

## Acceptance

- `container:action-trace` events are consumable by the frontend.
- Log panel can show action trace lines without a new panel.
- Structured trace entries remain available for a future viewer.
- Worktree is clean after commits.
