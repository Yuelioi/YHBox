# N3 — 子图生命周期闭环

## Goal

让用户能明确选择“复用同一定义”还是“创建独立副本”，并能安全展开调用、删除调用和物理删除定义。

## Status

In Progress

## Scope

- 复制 GraphCall：创建同一 Graph 定义的新调用，保留调用级 binding，生成独立调用 ID/位置。
- 复制 Graph 定义：深复制定义、生成全新 graph/element identity，并可选择立即创建指向新定义的调用。
- 展开 GraphCall：把被调用定义的节点、嵌套调用、注释和连线原子内联到 caller，正确重连入口/输入/输出/出口。
- 删除调用只删除实例；零引用定义可物理删除；有引用定义支持显式展示影响后原子删除所有调用与定义。
- 所有复合动作都是一个 undo 单位和一个 authoring command，不由 UI 拼接脆弱命令序列。

## Steps

1. 建立复制/展开/级联删除的 Source 转换失败测试，覆盖多 caller、嵌套调用、binding 和多 exits。
2. 扩展 authoring patch 与 EditorSession 复合命令，保持前后端同一语义与引用保护。
3. 在子图管理器和 GraphCall Inspector 提供无歧义动作及影响确认。
4. 覆盖保存重开、undo/redo、compile 与 `task check`。

## Verification

- 复制调用后两个调用引用同一 graph ID，但拥有独立 call ID、位置和 binding。
- 复制定义后修改副本不会影响原定义，内部所有 element ID 与嵌套引用保持有效且不冲突。
- 展开调用后外部入口、data input/output、exec/error exit 连线语义不变，原调用消失且定义仍存在。
- 有多个调用时定义删除默认拒绝；显式级联确认后一次 undo 可恢复全部调用和定义。
- 保存重开与 compiler 看到的 Source 拓扑一致，不产生第二套 runtime。

## References

- [Subgraph management research](../references/subgraph-management-research.md)
- [N1 definition management](stage-n1-subgraph-definition-management.md)
- [N2 interface editor](stage-n2-subgraph-interface-editor.md)
