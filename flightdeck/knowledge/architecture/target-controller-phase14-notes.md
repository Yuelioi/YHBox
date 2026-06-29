# Target / Controller Phase 14 Notes

SUMMARY: Phase 14 consumes action trace events in the frontend log store
READ WHEN: Building trace UI / debugging frontend trace visibility / changing log store event ingestion
RECHECK WHEN: `container:action-trace`, `frontend/src/lib/events.ts`, or `useLogStore` changes

---

Phase 14 adds the first frontend consumer for action traces:

- `frontend/src/lib/events.ts` subscribes to `container:action-trace`.
- `useLogStore` stores structured entries in `actionTraces`.
- The same action is appended to log lines with level `action` and source `CTR`.
- `clear()` clears both regular log lines and structured action traces.

Operational effect:

- Action traces are visible immediately in the existing log panel.
- Future trace UI can consume `useLogStore().actionTraces` without waiting for backend API changes.

Still not covered:

- Dedicated action trace/timeline panel.
- Persistence.
- Payload redaction/batching.
