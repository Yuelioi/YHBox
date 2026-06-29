# Phase 41 — App Runtime Uses Central Node Registration

## Problem

Phase 40 centralized full node registration for catalog, CLI, MCP, and guard tests, but the app entrypoint and one full runtime dispatch test still carried their own manual import lists. That leaves the release binary and runtime contract tests vulnerable to the same "new node package added, but one path forgot to import it" drift.

## Scope

- Replace the app entrypoint's manual built-in node imports with `_ "yotta/internal/nodes/all"`.
- Replace the full runtime dispatch test's manual built-in node imports with `_ "yotta/internal/nodes/all"`.
- Update catalog comments so callers know the intended full-registration import.

## Non-goals

- No node behavior changes.
- No registry API changes.
- No cleanup of intentionally scoped tests that import only the packages they exercise.
