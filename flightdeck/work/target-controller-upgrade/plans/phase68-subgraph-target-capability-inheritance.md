# Phase68 - subgraph target capability inheritance

## Why

Phase64-67 validate target capabilities inside one graph. Real containers often select a target in the main graph and then call a subgraph. If the subgraph contains target-aware nodes and no local target selection, those nodes run against the caller's active target. The validator should check that inherited target.

## Contract

- A `Subgraph` call inherits the nearest upstream target selection from its caller graph.
- A subgraph node with no local upstream target selection is validated against the inherited target.
- A local target selection inside the subgraph overrides the inherited target.
- Ambiguous caller target remains conservative and does not produce a static capability error.

## Tasks

- [x] Add failing test for `AndroidTarget -> Subgraph(MouseMoveRel)`.
- [x] Add inherited target support to target capability validator.
- [x] Keep local target selection override behavior.
- [x] Update Flightdeck notes.
- [x] Run verification and commit.
