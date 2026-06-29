# Node Step Debugging Design

## Goal

Add a practical node-graph debugging mode that lets users run workflows one runtime step at a time. The first version should make real automation flows easier to inspect without changing normal run semantics.

The main user problem is repeated setup during debugging. Users should not need to rerun an entire container just to inspect a middle section, and they should be able to see which runtime step ran, which exit fired, and what data or variables changed.

## Terminology

- **Run command**: the existing toolbar action that enqueues a normal container run.
- **Start node**: the graph node with kind `Start`.
- **Debug from entry**: a debug session seeded from the tokens produced by `Start.Done`.
- **Debug from here**: a debug session seeded from a selected node's first exec input.
- **Runtime step**: one execution of one queued runtime `ExecToken` at the current graph boundary.
- **Region node**: a node that executes an internal region through `RunRegion`, such as `Subgraph`, `CollapsedNode`, `Loop`, `Parallel`, or `Race`.

This terminology avoids using `Start` as a button name, node kind, and mode name at the same time.

## Scope

V1 includes:

- Start a debug session from the workflow entry node.
- Start a debug session from a selected node.
- Execute one queued runtime token with `Step`.
- Resume execution with `Continue`.
- Request a node-boundary pause with `Pause`.
- Stop and tear down the debug session.
- Highlight the next, running, last, and failed debug nodes in the editor.
- Show the last executed node, exit name, output data, variable snapshot, and queued tokens.

V1 does not include:

- Breakpoints.
- Explicit subgraph controls such as `Step into` or `Step over`.
- Running arbitrary selected node fragments.
- Mocking missing external data inputs.
- Rewinding or undoing side effects.
- Debugging listener-driven nodes such as `EventTick`.

## Key Semantics

Debug execution is real execution, not simulation. Nodes that click, type, start programs, stop apps, switch windows, write files, call network services, or operate Android devices will perform those actions. The UI must show this warning near debug controls and in the `Debug from here` confirmation path.

`Step` executes exactly one queued runtime `ExecToken` at the current graph boundary:

- The session takes the next token from its queue.
- It executes the node referenced by that token once.
- It records the fired exit and output data.
- It enqueues downstream tokens normally.
- It pauses before the next queued token.

This matches the existing token dispatch model and avoids inventing a second graph interpreter.

Important consequence: a single runtime step can include internal work inside a region node. In V1:

- `Subgraph` and `CollapsedNode` are atomic. A step on the call node runs the called subgraph to completion and returns the reached output.
- `Loop` is atomic at the outer graph boundary. A step on a loop node may execute multiple body iterations.
- `Parallel` and `Race` are atomic at the outer graph boundary according to current region-runner semantics.

This is not a contradiction with future `Step into` / `Step over` controls. V1 has only the default behavior: region nodes are stepped as one outer runtime step. V2 can add explicit controls to enter region internals.

When a normal node fans out to multiple downstream exec edges, each downstream token is stepped separately in FIFO order.

## Start Modes

### Debug From Entry

This behaves like a normal container run, but the session pauses before executing the first real token produced from `Start.Done`.

This is the safest default because setup nodes run in graph order.

### Debug From Here

This seeds the debug queue with the selected node's first exec input. The graph downstream from that node executes normally.

If the selected node depends on skipped context, the session exposes that missing context explicitly:

- Startup validation catches static errors before creating the session.
- Runtime-only context failures put the session into `failed` when the step reaches the affected node.
- The failed state remains inspectable until the user stops or restarts the session.

Examples:

- A click node may fail if no target node has run.
- A data input may fail if it depends on an upstream exec output not produced in this debug session.
- A variable read may use the container's initial/global value instead of a skipped upstream write.

The UI command label is `Debug from here`, not `Run this node`. The confirmation text should say that upstream nodes are skipped and existing variable values may be used.

## Backend Architecture

Add a debug execution path beside the existing normal queued run path. Do not overload the public `Run(containerID)` method.

### Service API

Service methods:

- `DebugStart(containerID string, options DebugStartOptions) (DebugSessionState, error)`
- `DebugStep(sessionID string) (DebugSessionState, error)`
- `DebugContinue(sessionID string) (DebugSessionState, error)`
- `DebugPause(sessionID string) (DebugSessionState, error)`
- `DebugStop(sessionID string) (DebugSessionState, error)`
- `DebugState(sessionID string) (DebugSessionState, error)`

`containerID` is the target container ID. It is a method argument instead of an option because every session belongs to exactly one container.

`DebugStartOptions`:

- `StartNodeID string`: optional. Empty means debug from entry; non-empty means debug from here.
- `GraphPath []string`: reserved for future subgraph-local starts. V1 rejects non-empty `GraphPath` with `debug_unsupported_graph_path`.

### Session State

`DebugSessionState`:

- `SessionID string`
- `ContainerID string`
- `Status string`: `paused`, `stepping`, `running`, `pause_requested`, `finished`, `failed`, `stopped`
- `Mode string`: `entry` or `from_node`
- `StartNodeID string`
- `CurrentNodeID string`: next queued token when paused
- `CurrentNodeKind string`
- `RunningNodeID string`: node currently executing while `stepping`, `running`, or `pause_requested`
- `RunningNodeKind string`
- `LastNodeID string`
- `LastNodeKind string`
- `LastExit string`
- `LastOutput map[string]any`
- `Vars map[string]any`
- `Queue []DebugTokenSummary`
- `Error *RunError`
- `Warnings []DebugWarning`

`DebugTokenSummary`:

- `NodeID string`
- `NodeKind string`
- `InPin string`
- `GraphPath []string`
- `LoopDepth int`
- `ExecDataKeys []string`

The frontend uses `Queue[0]` as the next paused node and may show the rest as a compact queue preview. It should not receive full `ExecData` values in queue summaries because that can be large and confusing.

`DebugWarning`:

- `Code string`
- `Message string`
- `NodeID string`
- `Params map[string]any`

V1 should emit at least one warning for `Debug from here`: skipped upstream nodes can make variables, active targets, and held exec outputs differ from a full run.

### Session Storage

Sessions are process-local in-memory state. There is no Redis, persistence, or recovery after app restart in V1.

The service keeps one active debug session at a time. If another client or editor tab starts a session while one exists, `DebugStart` returns `debug_session_busy`.

If the app process restarts, all sessions are lost. The frontend should clear local debug UI state when `DebugState` returns `debug_session_not_found`.

### Execution Exclusivity

Normal runs and debug runs must share one process-wide execution lease:

- A normal worker target run acquires the lease for the duration of that target run.
- A debug session acquires the lease in `DebugStart` and releases it only on `finished`, `failed` after `DebugStop`, or `stopped`.
- Normal Run is rejected while a debug session owns the lease.
- DebugStart is rejected while the normal worker is running.

This preserves the existing single-driver invariant for mouse, keyboard, windows, Android devices, and capture backends.

### Command Concurrency

Each debug session has a command mutex and a cancel context.

Allowed commands by state:

- `paused`: accepts `DebugStep`, `DebugContinue`, `DebugStop`.
- `stepping`: accepts `DebugStop`; another `DebugStep` or `DebugContinue` returns `debug_session_busy`.
- `running`: accepts `DebugPause` and `DebugStop`; `DebugStep` returns `debug_session_busy`.
- `pause_requested`: accepts `DebugStop`; another `DebugPause` is idempotent.
- `finished`, `failed`, `stopped`: accepts `DebugState`; start a new session to run again.

`DebugStep` and `DebugContinue` should start execution asynchronously and return the updated state immediately (`stepping` or `running`). Completion is delivered by events and can also be read with `DebugState`. This prevents long `Sleep`, wait, or automation nodes from leaving the UI waiting on a single RPC promise.

`DebugPause` does not interrupt a currently running node. It sets a boundary pause request. The session transitions to `paused` after the current node completes and before the next token executes.

`DebugStop` cancels the current node context and runs teardown. It should be callable while a long node is executing.

### Runner Integration

The existing `ContainerRunner` should gain a debug-capable dispatch primitive rather than duplicating graph interpretation.

Concrete runner split:

- Extract runtime setup into `StartRuntime(ctx)` and teardown into `StopRuntime()`.
- Add `SeedFromEntry()` to enqueue tokens from `Start.Done`.
- Add `SeedFromNode(nodeID string)` to enqueue one token for the selected node's first exec input.
- Add `StepOnce(ctx) (DebugStepResult, error)` to execute one queued token.
- Keep normal `Run(ctx)` behavior by calling the same setup, seed, and step primitives in a tight loop.

Normal `Run(ctx)` must remain covered by existing tests. New tests should assert that normal run behavior does not change after the split.

The debug session holds the runner instance between steps so state survives:

- active target/window
- variables
- stopwatch table
- held exec outputs
- loop stack in queued tokens
- runtime resources that need teardown on stop or finish

## Validation

Normal full-container validation remains unchanged.

For debug from entry, run normal validation before creating the session.

For debug from here, use relaxed debug validation before creating the session:

- The target node must exist in the main graph.
- The target node must have at least one exec input.
- Non-empty `GraphPath` is rejected with `debug_unsupported_graph_path`.
- Invalid pin references are startup errors.
- Missing node kinds are startup errors.
- Bad dynamic config is a startup error.
- Invalid regex config is a startup error.
- Missing templates and clips are startup errors.
- Static type errors are startup errors.
- `EventTick` in the runnable graph is a startup error with `debug_listener_unsupported`.
- Missing path from the Start node to `StartNodeID` is allowed.
- Missing Start node is allowed only for debug from here.

Runtime context gaps caused by skipping upstream execution are not startup validation errors unless they are statically provable. They fail at the step that needs them and transition the session to `failed`.

Examples of runtime failures:

- A target-aware node runs before any target has been selected.
- A data pull needs a held exec output that was never produced.
- A required literal/input is still empty at the selected start point.

The UI should present startup validation failures in the existing validation panel. Runtime debug failures should appear in the debug panel and focus the failed node.

## Frontend UX

### State Updates

Use Wails events for live updates and RPC for commands:

- Backend emits `debug:state` after every state transition.
- Backend emits existing `container:node-enter` for the node currently executing, or a debug-specific event with the same payload plus `sessionId`.
- Frontend calls `DebugState(sessionID)` on editor mount or reconnect to resync.

No polling loop is required in normal operation. A manual resync on editor open is enough for V1.

### Toolbar

Toolbar controls:

- `Run`: existing normal run command.
- `Debug`: starts debug from entry.
- `Step`: enabled only when session is `paused`.
- `Continue`: enabled only when session is `paused`.
- `Pause`: enabled only when session is `running`.
- `Stop`: enabled when session is `paused`, `stepping`, `running`, or `pause_requested`.

Disable normal Run while a debug session is active. If a normal run is active, disable Debug and show the existing running state.

### Node Context Menu

Add:

- `Debug from here`

When chosen, show a compact confirmation if the node is not the Start node:

- Upstream nodes are skipped.
- Existing variable values and active target state may be used.
- Actions are real and cannot be rewound.

### Canvas

Visual states:

- Next queued node while paused: strong debug outline.
- Currently executing node: existing running highlight plus debug accent.
- Last executed node: brief completion flash or subtle marker.
- Failed node: existing error styling and debug panel focus.

For region nodes, the outer region node is highlighted while its internal region runs in V1.

### Debug Panel

The debug panel can live in the inspector or bottom panel. It shows:

- Session status.
- Mode: entry or from here.
- Next node.
- Running node.
- Last node and exit.
- Last output data as compact JSON.
- Variable snapshot.
- Queue preview from `DebugTokenSummary`.
- Warnings and runtime failure details.

Copy should avoid pretending this is harmless. Use wording such as `Debug run executes real actions`.

## Error Handling

If startup validation fails, no session is created.

If a step fails at runtime, the session enters `failed` and remains inspectable. The user can stop it or start a new session after stopping.

If a node fires an error exit that is wired, that is not a failed debug session. It is a normal step result with `LastExit` set to that exit.

If the user stops the session, teardown must run:

- cancel the current node context
- release held input
- close input/capture backends
- stop any runtime resources
- release the execution lease
- clear active debug state

V1 rejects listener-driven debug sessions at startup validation. If debug validation sees `EventTick` in the runnable graph, it rejects the session with `debug_listener_unsupported`. Disabling listeners would make debug behavior differ from real behavior, so V1 does not do that.

## Edge Cases

### Long Wait Or Sleep Nodes

`Step` on a long `Sleep`, wait, visual detection, or external call can take time. The session enters `stepping`, the UI remains interactive, and `Stop` can cancel through the session context.

`Pause` cannot interrupt a node mid-execution. It only pauses before the next token.

### Side Effects

Debug actions are not reversible. File writes, network calls, input injection, app launches, app stops, and window operations stay applied even if the session is stopped later.

The UI must warn about this before debug from here and in the debug panel.

### Existing Variables In Debug From Here

If a skipped upstream node would normally write a variable, but the variable already has an initial/global value, V1 may use that value. This can make a debug-from-here path look successful while differing from a full run.

V1 handles this with a warning, not automatic upstream execution. Future work can add upstream dependency analysis or variable diffing.

### Multiple Clients

The app is a local desktop application, but multiple windows or clients may still call the service. V1 allows only one active debug session process-wide. Later commands must pass the session ID and get `debug_session_not_found` or `debug_session_busy` when appropriate.

### Restart

Sessions are in-memory only. App restart drops them. The frontend clears debug state when the backend cannot find the session.

## Testing Plan

Backend tests should use concrete API sequences and state assertions.

### Backend API Tests

Debug from entry:

1. Create container `Start -> SetVar -> Log`.
2. Call `DebugStart(containerID, DebugStartOptions{})`.
3. Expect state `paused`, `CurrentNodeID == setvar`.
4. Call `DebugStep(sessionID)`.
5. Expect immediate state `stepping`.
6. Wait for `debug:state` or call `DebugState` until paused.
7. Expect `LastNodeID == setvar`, `LastExit == Done`, variable snapshot contains the new value, and `CurrentNodeID == log`.

Debug from here:

1. Create container without a Start path to target node.
2. Call `DebugStart(containerID, DebugStartOptions{StartNodeID: target})`.
3. Expect state `paused`, `CurrentNodeID == target`, and warning code for skipped upstream context.

Invalid debug from here:

1. Unknown target node returns startup error `debug_invalid_start_node`.
2. Target node with no exec input returns `debug_start_node_not_executable`.
3. Non-empty `GraphPath` returns `debug_unsupported_graph_path`.

Runtime failure:

1. Debug from a target-aware node that needs a target but skips target setup.
2. Step it.
3. Expect session state `failed`, failed node ID set, and error envelope populated.

Stop and restart:

1. Start debug session.
2. Step once.
3. Stop.
4. Expect state `stopped`, execution lease released.
5. Start a new debug session for the same container.
6. Expect success.

Concurrency:

1. Start debug session.
2. Call `DebugStep`.
3. While state is `stepping`, call `DebugStep` again.
4. Expect `debug_session_busy`.
5. Call `DebugStop` and expect cancellation plus teardown.

Long node:

1. Use `Sleep` with a visible duration.
2. Step it.
3. Expect immediate state `stepping`.
4. Stop before the sleep completes.
5. Expect state `stopped` and no leaked session.

Region node:

1. Create `Start -> Subgraph -> Log`, where the subgraph writes a variable.
2. Step the Subgraph call.
3. Expect the subgraph completed as one outer step, the variable changed, and the next node is `Log`.

Normal run unchanged:

1. Run existing normal-run tests.
2. Add a focused test proving `Run(ctx)` still drains the same graph as before the stepping split.

### Frontend Tests

Controls:

1. No active session: show Run and Debug; hide/disable Step, Continue, Pause, Stop.
2. Paused session: enable Step, Continue, Stop; disable Run.
3. Running session: enable Pause and Stop; disable Step and Continue.
4. Stepping session: enable Stop; disable Step and Continue.

Context menu:

1. Right-click executable node.
2. Expect `Debug from here`.
3. Pick it and assert backend `DebugStart(containerID, { StartNodeID })` call.

State events:

1. Emit `debug:state` with `CurrentNodeID`.
2. Expect canvas debug highlight on that node.
3. Emit failed state.
4. Expect failed node styling and debug panel details.

### Integration Smoke

- Step through `Start -> SetVar -> Log`.
- Debug from a middle `Log` node and verify skipped-context warning.
- Step `Start -> Win32WindowTarget -> ClickAt` and confirm pause between target setup and click.
- Step a `Subgraph` call and confirm it is atomic in V1.
- Stop during `Sleep` and confirm the UI/session recovers.

## Future Work

V2:

- Node breakpoints.
- Explicit `Step over` control for region nodes.
- Explicit `Step into` control for subgraphs and loop bodies.
- Run selected fragment with strict single-entry validation.
- Manual mock values for external data inputs.
- Variable diff view.
- Persist last debug start node per container.

V3:

- Conditional breakpoints.
- Watch expressions.
- Time-travel-style snapshots for pure data and variables only.
- Debug recording export for bug reports.
