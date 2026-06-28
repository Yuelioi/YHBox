# Phase69 - collapsed node target capability inheritance

## Why

Phase68 propagates caller target selection into `Subgraph` calls. `CollapsedNode` also calls a backing subgraph through `SubgraphID`, so it should inherit the caller target for capability validation as well. Otherwise collapsing a graph changes static validation behavior.

## Contract

- `Subgraph` and `CollapsedNode` are both subgraph-call nodes for target capability inheritance.
- The existing local target override behavior remains unchanged.

## Tasks

- [x] Add failing test for `AndroidTarget -> CollapsedNode(MouseMoveRel)`.
- [x] Extend inherited target propagation to `CollapsedNode`.
- [x] Run verification and commit.
