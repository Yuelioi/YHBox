# Phase 42 — Node Registration Drift Guard

## Problem

`internal/nodes/all` reduces duplicate full-registration imports, but it is still a manual list. Without a guard, adding `internal/nodes/<new>` can still miss the central import and silently disappear from app startup, catalog, MCP, and runtime guard paths.

## Scope

- Add a test in `internal/nodes/all` that compares `doc.go` blank imports with the actual `internal/nodes/*` package directories.
- Move the container package's broad test setup import to `_ "yotta/internal/nodes/all"`.

## Non-goals

- No generated registration.
- No behavior changes to scoped tests that intentionally import only the node packages under test.
