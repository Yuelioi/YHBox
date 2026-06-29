# Index — node-step-debugging

## State

Design approved at concept level by user. External design feedback has been evaluated and incorporated into `design.md`. No implementation has started.

## Next

User review gate:
- Review the updated `design.md`, especially V1 token-step semantics and atomic region-node behavior.
- If approved, create an implementation plan before touching runtime/frontend code.

## Read now

- `design.md` — V1 node step debugging scope, runtime semantics, UX, validation, and test plan.

## Read if

- `flightdeck/work/window-control/design.md` — useful reference for how prior runtime-affecting node work was structured.
- `flightdeck/knowledge/build/build.md` — build and smoke-test expectations before shipping implementation.

## Progress

Done:
- Defined V1 around real token-step debugging: Debug from entry, Debug from here, Step, Continue, Pause, Stop.
- Clarified that V1 Step means one outer runtime `ExecToken`; `Subgraph`, `CollapsedNode`, `Loop`, `Parallel`, and `Race` are atomic region nodes in V1.
- Defined debug session DTOs, token summaries, warnings, in-memory storage, execution lease, command concurrency, and live update events.
- Defined startup validation vs runtime context failures, including `GraphPath`, invalid selected nodes, and listener-driven nodes.
- Added concrete backend, frontend, and integration test sequences.
- Deferred breakpoints, selected-fragment runs, explicit subgraph step into/over, and mock external inputs to later phases.
- Explicitly rejected `EventTick` listener debugging in V1.

Verified:
- Design self-review completed for terminology, scope, API, validation, edge cases, and test coverage.
