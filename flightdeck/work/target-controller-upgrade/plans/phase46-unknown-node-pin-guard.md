# Phase 46 — Unknown Node Pin Guard

## Problem

Even after rejecting missing VueFlow handles, `pinsFor(unknownKind)` still rendered fake lowercase `in` / `out` pins. That allowed unknown or registry-missing nodes to expose invalid handles, which could later become invalid persisted edges.

## Scope

- Make unknown node kinds render no pins by default.
- Preserve explicit virtual subgraph marker pins (`SubgraphInput.Done`, `SubgraphOutput.In`).
- Add a frontend regression test for both behaviors.

## Non-goals

- No compatibility fallback for unknown node kinds.
- No changes to registered node specs.
