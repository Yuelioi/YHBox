# Target / Controller Phase 8 Notes

SUMMARY: Phase 8 adds container/node/input-pin source metadata to runtime controller action traces
READ WHEN: Debugging trace attribution / extending trace UI or persistence / changing runtime input dispatch
RECHECK WHEN: `execNodeViaFramework`, `node.ServiceBundle`, `inputAdapter.controller`, or trace schema changes

---

Phase 8 adds source metadata to controller action traces:

- `trace.ActionSource` carries `ContainerID`, `NodeID`, `NodeKind`, and `InPin`.
- `trace.ActionRecord.Source` stores that source beside action request/result/status data.
- `execNodeViaFramework` and `execNodeAsRegionViaFramework` copy `node.ServiceBundle` per dispatched node and replace only `Input`.
- The source-aware `inputAdapter` wraps the runtime recorder before passing it to `Win32Controller`.

Design constraint:

- Do not store a mutable "current node" on `RuntimeContext`; dispatch paths can share runtime state across branches/listeners.
- Source metadata belongs to the per-node input service copy, not the shared runner bundle.

Operational effect:

- Nodes dispatched through the framework now emit input traces attributable to the container, node, node kind, and incoming exec pin.
- `ClickAt` produces both `move` and `click` trace records with the same source metadata.
- Direct `NewInputAdapter(rt)` usage keeps old behavior and records empty source metadata.

Still not covered:

- UI trace viewer and persistence.
- Drag, MouseDown, MouseUp, TypeText, Screenshot routing through `Win32Controller`.
- Rich pin/output/result lineage beyond the incoming exec pin.
