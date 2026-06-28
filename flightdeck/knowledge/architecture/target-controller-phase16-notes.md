# Target / Controller Phase 16 Notes

SUMMARY: Phase 16 adds a frontend action trace drawer backed by `useLogStore().actionTraces`
READ WHEN: Debugging click/input/capture execution traces from the UI
RECHECK WHEN: `container:action-trace`, `ActionTraceEntry`, or `LogPanel` changes

---

Phase 16 exposes the structured action trace cache in the frontend:

- `ActionTraceDrawer.vue` renders recent traces newest-first.
- `LogPanel` has a route/activity icon button that opens the trace drawer.
- The drawer shows action, status, source node, target, backend, duration, coordinate step count, error, and optional request/result payload.
- `zh.ts` and `en.ts` include matching `log.action_trace.*` keys.

Verification:

- `pnpm vue-tsc --noEmit`
- `pnpm i18n:check`
- `pnpm vitest run src/stores/__tests__/log.spec.ts src/stores/log.spec.ts`

Still not covered:

- Persistent run-history trace storage.
- Payload redaction rules for long-term storage.
- Browser/Android controller backends.
