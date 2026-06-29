# Phase 8 Trace Source Metadata Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Attach container/node/input-pin source metadata to controller action trace records.

**Architecture:** Add `trace.ActionSource` to `trace.ActionRecord`. Runtime wraps the per-run trace recorder with a small source-enriching recorder for each node execution. `execNodeViaFramework` passes a copied `ServiceBundle` with a source-aware `InputService`, avoiding shared mutable "current node" state and keeping parallel dispatch safe.

**Tech Stack:** Go, runtime dispatch, trace package, node service adapters, `go test`.

---

## Scope

In scope:

- Add `ActionSource` with `ContainerID`, `NodeID`, `NodeKind`, and `InPin`.
- Preserve old trace behavior when no source is provided.
- Add per-node source metadata for input actions run through normal `execNodeViaFramework`.
- Keep source metadata out of pure data evaluation and non-input services for now.

Out of scope:

- UI trace viewer.
- Pin/output result metadata.
- Region node body tracing beyond child nodes dispatched normally.
- Persistence.

## Design Notes

- Do not store current node source on `RuntimeContext`; parallel branches share it.
- Copy `node.ServiceBundle` per executed node and replace only `Input`.
- `inputAdapter` carries optional `trace.ActionSource`.
- `inputAdapter.controller()` passes a source-enriching recorder to `Win32Controller`.

## Tasks

1. Commit this plan and update `flightdeck/work/target-controller-upgrade/index.md`.
2. Add failing trace package test proving `ActionRecord.Source` can store source metadata.
3. Add `ActionSource` to `internal/automation/trace`.
4. Add failing runtime test proving `ClickAt` trace includes node source after `execNodeViaFramework`.
5. Implement source-aware input adapter and per-node service bundle copy.
6. Run verification, add Phase 8 notes, update index, and commit docs.

## Verification

```powershell
go test ./internal/automation/trace -count=1
go test ./internal/services/container/runtime -run "TestInputAdapter|TestTraceSource|TestClickAt|TestRuntimeContextTrace" -count=1
go test ./internal/automation/... -count=1
```

## Acceptance

- Direct `NewInputAdapter(rt)` calls still record actions without source metadata.
- Nodes dispatched through `execNodeViaFramework` produce trace records with source metadata.
- No global mutable current-node state is introduced.
- Worktree is clean after commits.
