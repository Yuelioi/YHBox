# Target / Controller Phase 1 Notes

SUMMARY: Phase 1 introduces Target and Controller types, wraps Win32 behavior, and keeps existing nodes compatible
READ WHEN: Continuing Target/Controller implementation / debugging why nodes still call old services / deciding when to migrate nodes to controller APIs
RECHECK WHEN: `internal/automation` or runtime service adapters change

---

Phase 1 is an adapter phase. It does not change stored container JSON, node specs, or frontend node UI.

New packages:

- `internal/automation/target`: target identity and coordinate-space value types.
- `internal/automation/controller`: controller capabilities and Win32 controller wrapper.

Runtime bridge:

- `internal/services/container/runtime/automation_adapters.go` maps `winutil.WindowHandle` to `target.Target`.

Compatibility rule:

- Existing runtime services remain authoritative for current nodes.
- New controller code wraps the same input/capture/window shape so behavior can be compared before node migration.
- Nodes still call the current `node.ServiceBundle` adapters; no graph behavior changes in Phase 1.

Next phase:

- Add trace records around controller calls.
- Route one narrow node path through the Win32 controller behind a feature flag.
- Use AE main-window to composition-dialog smoke as the first real target consistency test.
