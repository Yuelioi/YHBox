# Phase 40 — Central Node Registration Import

## Problem

Multiple guard tests and CLI/MCP helpers manually maintain blank imports for every node package. This already caused drift: some full-catalog paths missed newer packages, so new nodes could escape i18n/spec/catalog checks.

## Scope

- Add one `internal/nodes/all` package that imports every built-in node package for registration side effects.
- Replace full-registration consumers with `_ "yotta/internal/nodes/all"`.
- Keep narrowly scoped tests with explicit imports when they intentionally exercise a small set.

## Non-goals

- No node behavior changes.
- No registry API changes.
