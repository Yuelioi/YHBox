# Type-Aware Inline Node Menu Implementation Plan

> Historical reference only. The implementation evidence is preserved below, but its source paths,
> commands, and Flightdeck references describe the pre-3.1 editor and must not be treated as current
> execution instructions.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans or inline TDD to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. User instruction overrides default commit cadence: do not commit unless the user explicitly asks.

**Goal:** Make the “drag pin to empty canvas” node picker show candidates that match the dragged pin direction and type, across every supported pin type, while keeping the plain canvas add-node menu unchanged.

**Architecture:** Put candidate selection in a small pure TypeScript helper so the Vue menu and auto-wire logic share the same compatibility model. Use the existing `NodeKindSpec` metadata (`execIn`, `execOut`, `dataIn`, `dataOut`, `isPureData`, `isVisualOnly`, `excludeFromPalette`) and existing `pinTypeCompat` rules rather than per-node hardcoding or priority sorting.

**Tech Stack:** Vue 3, Vitest, `frontend/src/components/containers/nodeRegistry`, existing inline menu composable `frontend/src/composables/containerEditor/useInlineMenu.ts`.

---

## Scope

Implement now:

- Plain canvas menu keeps the existing palette behavior, including palette-eligible visual nodes such as `CommentBox`.
- Data output drag shows nodes with at least one compatible data input.
- Data input reverse-drag shows nodes with at least one compatible data output.
- Data candidate filtering is strict: exact type or `any` only. Warning-only coercions such as `number -> string`, `number -> bool`, or `bool -> number` remain valid at the connection layer but must not broaden the inline node picker.
- Exec output drag (`Done`, `Fail`, etc.) shows executable nodes with exec input pins; it excludes pure data nodes, parameter conversion/purefunc nodes, visual nodes, and palette-excluded marker nodes.
- Exec input reverse-drag shows executable nodes with exec output pins; same exclusions as exec output.
- All current frontend pin types are covered in tests: `number`, `bool`, `string`, `point`, `any`, `list`, `file`.
- No node priority sorting in this topic.

Defer:

- Ranking “best” nodes first.
- Error-specific recommendations for `Fail` such as log/notify/retry.
- Per-node semantic tags such as `text-transform`, `file-source`, `http-action`.

## File Map

- Create: `frontend/src/components/containers/inlineNodeCandidates.ts`
  - Pure helper for filtering insertion candidates.
- Create: `frontend/src/components/containers/inlineNodeCandidates.test.ts`
  - Focused tests for plain menu, exec drag, data drag, and every pin type.
- Modify: `frontend/src/components/containers/InlineContextMenu.vue`
  - Delegate candidate filtering to the helper.
  - Keep data-pin header text only for typed data contexts; exec contexts show the normal add-node header.
- Modify: `frontend/src/composables/containerEditor/useInlineMenu.ts`
  - Pass exec pin context into the menu.
  - Use `pinTypeCompat` for data auto-wire compatibility.
- Modify: `frontend/src/composables/containerEditor/useInlineMenu.test.ts`
  - Prove exec pin drags carry context for menu filtering.
- Modify: `flightdeck/cockpit.md`
  - Track this work topic and verification status.

## Task 1: Extract Candidate Filter Helper

**Files:**
- Create: `frontend/src/components/containers/inlineNodeCandidates.ts`
- Create: `frontend/src/components/containers/inlineNodeCandidates.test.ts`
- Modify: `frontend/src/components/containers/InlineContextMenu.vue`

- [x] **Step 1: Write failing tests for candidate filtering**

Cover:

- Plain canvas menu excludes `excludeFromPalette` but keeps palette-eligible visual nodes.
- Data output pins find nodes with compatible `dataIn`.
- Data input pins find nodes with compatible `dataOut`.
- Exec output pins find executable nodes with `execIn` and exclude pure data/visual/marker nodes.
- Exec input pins find executable nodes with `execOut` and exclude pure data/visual/marker nodes.

- [x] **Step 2: Verify RED**

Run: `cd frontend && pnpm exec vitest run src/components/containers/inlineNodeCandidates.test.ts`

Expected: fail because `inlineNodeCandidates.ts` does not exist.

- [x] **Step 3: Implement minimal helper**

Create `filterInlineNodeCandidates(specs, ctx)` using `pinTypeCompat`.

- [x] **Step 4: Wire menu to helper**

Replace the local filtering logic in `InlineContextMenu.vue` with `filterInlineNodeCandidates(allSpecs(), props.pinContext)`.

- [x] **Step 5: Verify GREEN**

Run: `cd frontend && pnpm exec vitest run src/components/containers/inlineNodeCandidates.test.ts`

Expected: pass.

## Task 2: Pass Exec Context From Pin Drag

**Files:**
- Modify: `frontend/src/composables/containerEditor/useInlineMenu.ts`
- Modify: `frontend/src/composables/containerEditor/useInlineMenu.test.ts`

- [x] **Step 1: Write failing test**

Add a test proving an exec pin drag to empty canvas opens the menu with `{ side: 'output', isExec: true }`.

- [x] **Step 2: Verify RED**

Run: `cd frontend && pnpm exec vitest run src/composables/containerEditor/useInlineMenu.test.ts -t "拖 exec pin"`

Expected: fail because exec drags previously used `pinContext: undefined`.

- [x] **Step 3: Implement**

Set exec drag context:

```ts
pinContext: isExec
  ? { side: startCopy.handleType === 'source' ? 'output' : 'input', isExec: true }
  : { pinType: pinType as PinType, side: startCopy.handleType === 'source' ? 'output' : 'input' }
```

- [x] **Step 4: Verify GREEN**

Run: `cd frontend && pnpm exec vitest run src/composables/containerEditor/useInlineMenu.test.ts`

Expected: pass.

## Task 3: Cover Every Pin Type

**Files:**
- Modify: `frontend/src/components/containers/inlineNodeCandidates.test.ts`
- Modify if needed: `frontend/src/components/containers/inlineNodeCandidates.ts`

- [x] **Step 1: Add table-driven tests for all data pin types**

Add cases for:

- `number`
- `bool`
- `string`
- `point`
- `any`
- `list`
- `file`

Each case must check both directions:

- output drag: source type -> candidate `dataIn`
- input reverse-drag: candidate `dataOut` -> target type

- [x] **Step 2: Verify RED/GREEN**

Run: `cd frontend && pnpm exec vitest run src/components/containers/inlineNodeCandidates.test.ts`

Expected: pass if helper is already generic; fail only if a type was missed.

- [x] **Step 3: Fix helper if a type fails**

Keep the helper generic. Do not add per-type branches unless compatibility rules require it.

- [x] **Step 4: Add regression tests for warning-only coercions**

Add tests proving `number` output drag does not include `string` / `bool` consumers, and `bool` input reverse-drag does not include `number` / `string` producers just because those edges are technically connectable with a warning.

## Task 4: Verify Auto-Wire Compatibility Uses The Same Type Rules

**Files:**
- Modify: `frontend/src/composables/containerEditor/useInlineMenu.test.ts`
- Modify if needed: `frontend/src/composables/containerEditor/useInlineMenu.ts`

- [x] **Step 1: Add tests for non-string auto-wire compatibility**

Add focused tests or extend existing setup so at least `number`, `point`, and `file` data pins use `pinTypeCompat` when choosing the first compatible input/output pin after menu selection.

- [x] **Step 2: Verify**

Run: `cd frontend && pnpm exec vitest run src/composables/containerEditor/useInlineMenu.test.ts`

Expected: pass with current generic `pinTypeCompat` usage.

## Task 5: Final Verification

**Files:**
- Modify: `flightdeck/work/type-aware-inline-node-menu/plan.md`
- Modify: `flightdeck/cockpit.md`

- [x] **Step 1: Run focused frontend tests**

Run:

```powershell
cd frontend
pnpm exec vitest run src/components/containers/inlineNodeCandidates.test.ts src/composables/containerEditor/useInlineMenu.test.ts src/components/containers/nodeRegistry/file-pin.test.ts
```

- [x] **Step 2: Run typecheck**

Run:

```powershell
cd frontend
pnpm typecheck
```

- [x] **Step 3: Update Flightdeck status**

Mark this topic complete in `flightdeck/cockpit.md` only after the focused tests and typecheck pass.

- [x] **Step 4: Leave uncommitted unless user asks**

Do not commit this topic unless the user explicitly asks.
