# Target Controller Upgrade — Phase 69 Notes

SUMMARY: CollapsedNode now inherits caller target selection for target capability validation
READ WHEN: 改 CollapsedNode、折叠/展开逻辑、匿名后备子图、target capability validator 时
RECHECK WHEN: CollapsedNode 不再以 SubgraphID 调用后备子图，或折叠节点获得独立 target 参数时

---

## Completed

- Target capability inheritance now treats both `Subgraph` and `CollapsedNode` as subgraph-call nodes.
- `AndroidTarget -> CollapsedNode(backing subgraph contains MouseMoveRel)` now reports `UNSUPPORTED_TARGET_CAPABILITY`.
- Existing `Subgraph` inheritance and local target override behavior remains covered by tests.

## Verification

- `go test ./internal/services/container -run "TestValidate_(CollapsedNodeInheritsAndroidTargetCapabilities|SubgraphInheritsAndroidTargetCapabilities|SubgraphLocalWin32TargetOverridesInheritedAndroidTarget)" -count=1`
