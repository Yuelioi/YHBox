# Node Step Debugging Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship V1 node step debugging with debug-from-entry, debug-from-here, Step, Continue, Pause, Stop, live state events, and editor highlights.

**Architecture:** Keep normal `Run(ctx)` semantics by extracting reusable runtime lifecycle and queue stepping primitives from `ContainerRunner`. Expose debug through `container.Service` as delegated RPC methods so the service package does not import the runtime package. A process-local debug manager owns one active runner and shares exclusivity with the normal worker.

**Tech Stack:** Go runtime/services, Wails 3 bindings, Vue 3/Pinia frontend, Vitest, Go test.

**Status:** Implemented. Runtime stepping, service/RPC layer, debug manager, frontend controls, right-click Debug from here, debug event store, and node highlights are in place.

**Verified:**
- `go test ./internal/services/container/runtime -run TestDebug -count=1`
- `go test ./internal/services/container -run TestServiceDebug -count=1`
- `go test . -run TestDebugManager -count=1`
- `go test ./...`
- `pnpm exec vitest run src/stores/execution.debug.test.ts`
- `pnpm exec vitest run`
- `pnpm exec vue-tsc --noEmit`
- `pnpm exec node src/i18n/check.cjs`
- `wails3 generate bindings ./...`
- `pnpm exec vite build --mode production`

---

## File Map

- Modify `internal/services/container/runtime/runner.go`
  - Add persistent debug queue on `ContainerRunner`.
  - Add `StartRuntime`, `StopRuntime`, `SeedFromEntry`, `SeedFromNode`, `StepOnce`, `QueueSnapshot`, and state snapshot helpers.
  - Rebuild existing `Run(ctx)` on top of the new primitives.
- Create `internal/services/container/runtime/debug.go`
  - Runtime-facing debug DTO aliases and `DebugStepResult` helpers if the runner code becomes too large.
- Create `internal/services/container/debug_types.go`
  - Public Wails-facing DTOs: `DebugStartOptions`, `DebugSessionState`, `DebugTokenSummary`, `DebugWarning`.
- Modify `internal/services/container/service.go`
  - Extend `Runner` with debug methods.
  - Add `DebugStart`, `DebugStep`, `DebugContinue`, `DebugPause`, `DebugStop`, `DebugState`.
- Modify `wire_container.go`
  - Extend `containerRunnerAdapter` with a debug manager, normal-run busy checks, and debug method implementations.
- Modify `main.go`
  - Construct the debug manager with the same runtime dependencies as normal runs.
  - Emit `debug:state`.
- Create `wire_container_debug_test.go`
  - Unit-test debug manager behavior without Wails.
- Create or modify runtime tests under `internal/services/container/runtime/`
  - Test one-token stepping, debug-from-node seed, unknown node failure, and region atomic behavior.
- Modify `frontend/src/lib/backend.ts`
  - Add debug DTO interfaces and `backend.containers.debug*` wrappers.
- Modify generated bindings under `frontend/bindings/yotta/internal/services/container/`
  - Prefer regenerating with Wails. If unavailable, add minimal generated JS/model entries matching existing format.
- Modify `frontend/src/stores/execution.ts`
  - Track debug session state and subscribe to `debug:state`.
- Modify `frontend/src/components/containers/ContainerEditorToolbar.vue`
  - Add Debug, Step, Continue, Pause, Stop controls with state gating.
- Modify `frontend/src/components/containers/menus/NodeContextMenu.vue`
  - Add `Debug from here` action.
- Modify `frontend/src/views/ContainerEditorView.vue`
  - Wire toolbar/menu actions to backend debug calls.
- Modify `frontend/src/components/containers/ContainerFlowNode.vue`
  - Add debug node visual states for next, running, last, failed.
- Modify `frontend/src/i18n/zh.ts` and `frontend/src/i18n/en.ts`
  - Add toolbar/menu/debug copy.

## Task 1: Runtime Step Primitives

**Files:**
- Modify: `internal/services/container/runtime/runner.go`
- Test: `internal/services/container/runtime/debug_step_test.go`

- [ ] **Step 1: Write failing test for stepping one token**

Create `internal/services/container/runtime/debug_step_test.go` with a test container `Start -> SetVar -> Log`. The test should call:

```go
if err := r.StartRuntime(ctx); err != nil { t.Fatal(err) }
defer r.StopRuntime()
if err := r.SeedFromEntry(); err != nil { t.Fatal(err) }
res, err := r.StepOnce(ctx)
```

Expected assertions:

```go
if res.NodeID != "set" { t.Fatalf("node = %q", res.NodeID) }
if res.Exit != "Done" { t.Fatalf("exit = %q", res.Exit) }
if got := r.QueueSnapshot()[0].NodeID; got != "log" { t.Fatalf("next = %q", got) }
```

- [ ] **Step 2: Verify red**

Run:

```powershell
go test ./internal/services/container/runtime -run TestDebugStepOnce -count=1
```

Expected: compile failure because `StartRuntime`, `SeedFromEntry`, `StepOnce`, and `QueueSnapshot` do not exist.

- [ ] **Step 3: Implement minimal runner primitives**

Add queue fields to `ContainerRunner`:

```go
queue []ExecToken
runtimeStarted bool
```

Add methods:

```go
func (r *ContainerRunner) StartRuntime(ctx context.Context) error
func (r *ContainerRunner) StopRuntime()
func (r *ContainerRunner) SeedFromEntry() error
func (r *ContainerRunner) SeedFromNode(nodeID string) error
func (r *ContainerRunner) StepOnce(ctx context.Context) (DebugStepResult, error)
func (r *ContainerRunner) QueueSnapshot() []ExecToken
func (r *ContainerRunner) VarSnapshot() map[string]any
```

`Run(ctx)` should call `StartRuntime`, `SeedFromEntry`, and repeatedly `StepOnce(ctx)` until queue is empty. Listener startup remains normal-run only.

- [ ] **Step 4: Verify green**

Run:

```powershell
go test ./internal/services/container/runtime -run TestDebugStepOnce -count=1
```

Expected: PASS.

- [ ] **Step 5: Add debug-from-node and region atomic tests**

Add tests:

- `TestDebugSeedFromNodeQueuesSelectedNode`
- `TestDebugStepSubgraphIsAtomic`
- `TestDebugSeedFromNodeRejectsMissingOrNonExecutableNode`

Run the same package test command and make each new test fail before adding the corresponding implementation.

## Task 2: Public Debug DTOs and Service Delegation

**Files:**
- Create: `internal/services/container/debug_types.go`
- Modify: `internal/services/container/service.go`
- Test: `internal/services/container/service_debug_test.go`

- [ ] **Step 1: Write failing service test**

Create a fake `Runner` implementing:

```go
DebugStart(id string, options DebugStartOptions) (DebugSessionState, error)
DebugStep(sessionID string) (DebugSessionState, error)
DebugContinue(sessionID string) (DebugSessionState, error)
DebugPause(sessionID string) (DebugSessionState, error)
DebugStop(sessionID string) (DebugSessionState, error)
DebugState(sessionID string) (DebugSessionState, error)
```

Assert `Service.DebugStart("c1", DebugStartOptions{StartNodeID: "n1"})` delegates `containerID` and options exactly.

- [ ] **Step 2: Verify red**

Run:

```powershell
go test ./internal/services/container -run TestServiceDebugStartDelegates -count=1
```

Expected: compile failure because debug DTOs and methods do not exist.

- [ ] **Step 3: Add DTOs and service methods**

Define exactly the DTO names from `design.md` in `debug_types.go`. Extend `Runner` and add `Service` methods that check container existence where applicable and delegate to `runner`.

- [ ] **Step 4: Verify green**

Run:

```powershell
go test ./internal/services/container -run TestServiceDebug -count=1
```

Expected: PASS.

## Task 3: Debug Manager and Execution Exclusivity

**Files:**
- Modify: `wire_container.go`
- Modify: `main.go`
- Test: `wire_container_debug_test.go`

- [ ] **Step 1: Write failing debug manager tests**

Test cases:

- `DebugStart` returns `debug_run_busy` when `worker.IsRunning()` is true.
- Second `DebugStart` returns `debug_session_busy`.
- `DebugStep` returns immediate state `stepping`, then emits paused state after the step.
- `DebugStop` releases the active session so a new start succeeds.

- [ ] **Step 2: Verify red**

Run:

```powershell
go test . -run TestDebugManager -count=1
```

Expected: compile failure or failing assertions because debug manager does not exist.

- [ ] **Step 3: Implement debug manager**

Add a process-local manager owned by `containerRunnerAdapter`. It constructs `RuntimeContext` and `ContainerRunner` using the same factory code as normal `runFunc`. It owns:

```go
mu sync.Mutex
session *debugSession
emit func(string, any)
workerBusy func() bool
```

It starts `DebugStep` and `DebugContinue` in goroutines and emits `debug:state` after every transition.

- [ ] **Step 4: Verify green**

Run:

```powershell
go test . -run TestDebugManager -count=1
```

Expected: PASS.

## Task 4: Frontend Debug Store and API

**Files:**
- Modify: `frontend/src/lib/backend.ts`
- Modify: `frontend/src/stores/execution.ts`
- Test: `frontend/src/stores/execution.debug.test.ts`

- [ ] **Step 1: Write failing store test**

Emit a fake `debug:state` event and assert the store records `debug.active`, `debug.status`, `debug.currentNodeID`, `debug.lastNodeID`, and `debug.failedNodeID`.

- [ ] **Step 2: Verify red**

Run:

```powershell
cd frontend; pnpm test src/stores/execution.debug.test.ts
```

Expected: FAIL because debug state does not exist.

- [ ] **Step 3: Add frontend debug types/store state**

Add `DebugSessionState`, `DebugStartOptions`, `DebugTokenSummary`, and `DebugWarning` interfaces in `backend.ts`. Add `backend.containers.debugStart/debugStep/debugContinue/debugPause/debugStop/debugState` wrappers. Extend `execution` store to consume `debug:state`.

- [ ] **Step 4: Verify green**

Run:

```powershell
cd frontend; pnpm test src/stores/execution.debug.test.ts
```

Expected: PASS.

## Task 5: Editor Controls and Highlighting

**Files:**
- Modify: `frontend/src/components/containers/ContainerEditorToolbar.vue`
- Modify: `frontend/src/components/containers/menus/NodeContextMenu.vue`
- Modify: `frontend/src/views/ContainerEditorView.vue`
- Modify: `frontend/src/components/containers/ContainerFlowNode.vue`
- Modify: `frontend/src/i18n/zh.ts`
- Modify: `frontend/src/i18n/en.ts`

- [ ] **Step 1: Add minimal component tests if existing harness supports it**

If the project lacks component mount tests for toolbar/menu, add pure helper tests for state gating instead of brittle DOM tests.

- [ ] **Step 2: Add toolbar actions**

Add Debug, Step, Continue, Pause, Stop buttons. Disable normal Run while debug is active. The toolbar should call:

```ts
backend.containers.debugStart(containerID, {})
backend.containers.debugStep(sessionID)
backend.containers.debugContinue(sessionID)
backend.containers.debugPause(sessionID)
backend.containers.debugStop(sessionID)
```

- [ ] **Step 3: Add node context action**

Add menu action `debug-from-here`; `ContainerEditorView` calls:

```ts
backend.containers.debugStart(containerID, { startNodeID: node.id })
```

Show a compact confirmation before starting from a non-Start node.

- [ ] **Step 4: Add visual states**

`ContainerFlowNode` should compute debug state from `useExecutionStore()`:

- next queued: `is-debug-next`
- running: existing `is-running`
- last: `is-debug-last`
- failed: `is-debug-failed`

- [ ] **Step 5: Verify frontend**

Run:

```powershell
cd frontend; pnpm typecheck
cd frontend; pnpm test
```

Expected: both pass.

## Task 6: Bindings, Full Verification, and Commit

**Files:**
- Modify generated Wails bindings if generation changes them.
- Modify `flightdeck/work/node-step-debugging/index.md`

- [ ] **Step 1: Regenerate Wails bindings**

Try:

```powershell
wails3 generate bindings -config .\build\config.yml
```

If unavailable, inspect generated binding format and patch only the new container service methods/models.

- [ ] **Step 2: Run targeted Go tests**

Run:

```powershell
go test ./internal/services/container/runtime ./internal/services/container ./internal/services/execution
go test . -run TestDebugManager -count=1
```

Expected: PASS.

- [ ] **Step 3: Run frontend checks**

Run:

```powershell
cd frontend; pnpm typecheck
cd frontend; pnpm test
```

Expected: PASS.

- [ ] **Step 4: Build smoke**

Run:

```powershell
go test ./...
cd frontend; pnpm build:dev
```

Expected: PASS. If full `go test ./...` hits known slow/integration failures, record exact failing package and rerun the targeted package set.

- [ ] **Step 5: Update work index and commit**

Update `flightdeck/work/node-step-debugging/index.md` with implementation progress and commit:

```powershell
git add .
git commit -m "feat(debug): add node step debugging"
```
