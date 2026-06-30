# fishing-v2 Rebuild Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild the local `bin/data/containers/fishing-v2/` container as a current package-layout automatic fishing container.

**Architecture:** Use a temporary Go generator that imports the real `container.Store`, `SubgraphStore`, validator, dependency closure, and asset store. The generator treats the old `container.json` as a business-flow source, patches current data structures in memory, writes through `Store.Save`, and then removes obsolete test container directories.

**Tech Stack:** Go 1.25, Yotta container service models, current node registry (`internal/nodes/all`), PowerShell verification.

---

### Task 1: Add A Temporary Rebuild Generator

**Files:**
- Create: `flightdeck/work/fishing-v2-rebuild/rebuild_fishing_v2.go`
- Modify: none

- [x] **Step 1: Write the generator**

Create `flightdeck/work/fishing-v2-rebuild/rebuild_fishing_v2.go` with a `package main` program that:

- reads `bin/data/containers/fishing-v2/container.json`;
- unmarshals into `container.Container`;
- rewrites main graph node kind `WindowTarget` to `Win32WindowTarget`;
- sets `Graph.SchemaVersion = container.GraphSchemaVersion`;
- adds state case `END` and edge `swState.END -> stop.In`;
- adds var `autoSellWhenNoCurrency` default `true`;
- loads `bin/data/subgraphs` with `container.NewSubgraphStore`;
- patches `state_BUYBAIT` via helper `patchBuyBait`;
- loads `bin/data/containers` with `container.NewStore`;
- injects a subgraph resolver using `dependency.Closure`;
- saves `fishing-v2` through `Store.Save`;
- validates via `container.NewService(...).ValidateContainerByID`;
- removes all sibling container directories except `fishing-v2`;
- writes a backup of the old `container.json` to `bin/data/_backups/fishing-v2-container.json` before saving.

- [x] **Step 2: Run compile check**

Run:

```powershell
go run .\flightdeck\work\fishing-v2-rebuild\rebuild_fishing_v2.go -dry-run
```

Expected:

- exits 0;
- prints that `fishing-v2` can be rebuilt;
- does not modify `bin/data/containers`.

### Task 2: Implement BUYBAIT No-Currency Branch

**Files:**
- Modify via generator: `bin/data/subgraphs/sg-aa7c1a8d-3cf6-4599-a3a0-d37c44863d5a.json`

- [x] **Step 1: Patch graph in generator**

In `patchBuyBait`, modify subgraph `sg-aa7c1a8d-3cf6-4599-a3a0-d37c44863d5a`:

- remove edge `sleepAfterBuyBtn.Done -> clickBuyConfirm.In`;
- add `CheckTemplate` node `checkNoCurrency` for template GUID `ffcf7913-e5fe-484a-9dca-589f2d9b9805`;
- add pure `GetVar` node `getAutoSellWhenNoCurrency` reading global `autoSellWhenNoCurrency`;
- add `If` node `ifAutoSellWhenNoCurrency`;
- add `SetVar` nodes:
  - `setStateShopSellNoCurrency`: `state = SHOPSELL`;
  - `setStateEndNoCurrency`: `state = END`;
- add edges:
  - `sleepAfterBuyBtn.Done -> checkNoCurrency.In`;
  - `checkNoCurrency.NotFound -> clickBuyConfirm.In`;
  - `checkNoCurrency.Found -> ifAutoSellWhenNoCurrency.In`;
  - `getAutoSellWhenNoCurrency.Value -> ifAutoSellWhenNoCurrency.Condition`;
  - `ifAutoSellWhenNoCurrency.True -> setStateShopSellNoCurrency.In`;
  - `ifAutoSellWhenNoCurrency.False -> setStateEndNoCurrency.In`;
  - both `SetVar.Done` outputs to `sgout.In`.

- [x] **Step 2: Verify the patched subgraph validates**

Run the generator dry-run again:

```powershell
go run .\flightdeck\work\fishing-v2-rebuild\rebuild_fishing_v2.go -dry-run
```

Expected:

- exits 0;
- validation output has no error severity entries.

### Task 3: Rebuild Local Container Data

**Files:**
- Modify generated local data: `bin/data/containers/fishing-v2/`
- Delete local data: sibling dirs under `bin/data/containers/`

- [x] **Step 1: Run the generator for real**

Run:

```powershell
go run .\flightdeck\work\fishing-v2-rebuild\rebuild_fishing_v2.go
```

Expected:

- `bin/data/containers/fishing-v2/package.json` exists;
- `bin/data/containers/fishing-v2/graph.json` exists;
- `bin/data/containers/fishing-v2/installation.json` exists;
- `bin/data/containers/fishing-v2/yotta-lock.json` exists;
- `bin/data/containers/fishing-v2/container.json` is gone;
- only `fishing-v2` remains under `bin/data/containers`.

- [x] **Step 2: Inspect key JSON facts**

Run:

```powershell
Get-ChildItem .\bin\data\containers | Select-Object Name
Get-Content .\bin\data\containers\fishing-v2\package.json
Get-Content .\bin\data\containers\fishing-v2\graph.json
Get-Content .\bin\data\containers\fishing-v2\installation.json
Get-Content .\bin\data\containers\fishing-v2\yotta-lock.json
```

Expected:

- package display name is `自动钓鱼`;
- graph contains `Win32WindowTarget`;
- graph does not contain `"version"` or `"WindowTarget"`;
- installation has `targetBindings.game`;
- lock contains template and subgraph dependencies.

### Task 4: Verify And Clean Up

**Files:**
- Delete: `flightdeck/work/fishing-v2-rebuild/rebuild_fishing_v2.go`
- Modify: `flightdeck/work/fishing-v2-rebuild/index.md`
- Modify: `flightdeck/cockpit.md`

- [x] **Step 1: Run backend verification**

Run:

```powershell
go test ./internal/services/container ./internal/services/container/dependency
```

Expected: PASS.

- [x] **Step 2: Delete the temporary generator**

Remove `flightdeck/work/fishing-v2-rebuild/rebuild_fishing_v2.go` after successful local data generation.

- [x] **Step 3: Update Flightdeck status**

Update `flightdeck/work/fishing-v2-rebuild/index.md` to say local data was rebuilt, test containers were removed, and manual game smoke is next.

Update `flightdeck/cockpit.md` to move `fishing-v2` from design-approved to rebuilt-awaiting-smoke.

- [ ] **Step 4: Commit tracked documentation changes**

Run:

```powershell
git add flightdeck/work/fishing-v2-rebuild/plan.md flightdeck/work/fishing-v2-rebuild/index.md flightdeck/cockpit.md flightdeck/work/fishing-v2-rebuild/design.md
git commit -m "docs(fishing): plan v2 container rebuild"
```

Expected: commit succeeds. `bin/` remains ignored, so local container data changes are verified but not committed.

## Self-Review

- Spec coverage: covers new package layout, old data cleanup, no-currency branch, user variable, validation, and manual smoke expectations.
- Placeholder scan: no TBD/TODO placeholders.
- Type consistency: uses current node kinds and pin names: `Win32WindowTarget`, `CheckTemplate.Found/NotFound`, `If.True/False`, `SetVar.Done`, `Subgraph` dynamic `done`, `Switch.END`.
