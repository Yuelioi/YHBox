# Target Controller Upgrade — Phase 68 Notes

SUMMARY: target capability validation now propagates caller target selections into called subgraphs
READ WHEN: 改 Subgraph 调用、active target 继承、target capability validator、子图运行语义时
RECHECK WHEN: 子图支持多入口 target 参数、图路径静态分析变成 branch-sensitive、Subgraph/CollapsedNode contract 变化时

---

## Completed

- `Subgraph` call now inherits the nearest upstream target selection from its caller graph for static capability validation.
- If a subgraph target-aware node has no local upstream target selection, validator checks it against the inherited target.
- Local target selection inside the subgraph overrides the inherited target.
- Nested subgraph calls propagate the inherited target and use a `(subgraphID,targetKind)` seen guard to avoid recursion loops.

## Boundary

Ambiguous upstream target selection remains conservative: if the caller graph cannot identify one unique nearest target, validator skips the inherited capability error and runtime controller checks remain the backup.

This phase handles `Subgraph`; `CollapsedNode` and script-driven `Subgraph({SubgraphID})` are still separate contracts.

## Verification

- `go test ./internal/services/container -run "TestValidate_Subgraph(InheritsAndroidTargetCapabilities|LocalWin32TargetOverridesInheritedAndroidTarget)" -count=1`
