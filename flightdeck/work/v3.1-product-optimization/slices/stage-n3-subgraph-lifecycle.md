# N3 — 子图生命周期闭环

## Goal

让用户能明确选择“复用同一定义”还是“创建独立副本”，并能安全展开调用、删除调用和物理删除定义。

## Status

Finished

## Scope

- 复制 GraphCall：创建同一 Graph 定义的新调用，保留调用级 binding，生成独立调用 ID/位置。
- 复制 Graph 定义：深复制定义并生成全新 graph identity；内部 element ID 保持 graph-scoped，因此可整体保留并继续
  维持内部端点与嵌套引用。
- 展开 GraphCall：把被调用定义的节点、嵌套调用、注释和连线原子内联到 caller，正确重连入口/输入/输出/出口。
- 删除调用只删除实例；零引用定义可物理删除；有引用定义支持显式展示影响后原子删除所有调用与定义。
- 所有复合动作都是一个 undo 单位和一次原子 authoring patch transaction，不由 UI 组件拼接脆弱命令序列。

## Steps

1. [x] 建立复制/展开/级联删除的 Source 转换测试，覆盖多 caller、嵌套调用、binding 和多 exits。
2. [x] 扩展 authoring patch 与 EditorSession 复合命令，保持前后端同一语义与引用保护。
3. [x] 在子图管理器和 GraphCall Inspector 提供无歧义动作及影响确认。
4. [x] 覆盖保存协议、undo/redo、后端原子应用与 `task check`；真实保存重开、compile/run 和 WebView 旅程归入 N4。

## Verification

- 复制调用后两个调用引用同一 graph ID，但拥有独立 call ID、位置和 binding。
- 复制定义后修改副本不会影响原定义，内部所有 element ID 与嵌套引用保持有效且不冲突。
- 展开调用后外部入口、data input/output、exec/error exit 连线语义不变，原调用消失且定义仍存在。
- 有多个调用时定义删除默认拒绝；显式级联确认后一次 undo 可恢复全部调用和定义。
- 保存重开与 compiler 看到的 Source 拓扑一致，不产生第二套 runtime。

## Result

- GraphCall Inspector 提供“复制调用 / 分叉为独立定义 / 展开调用”；子图管理器提供“复制定义 / 零引用删除 /
  显式级联删除”，调用删除继续只影响实例。
- 展开转换重新映射节点、嵌套调用、注释和边，重接单入口、data ports 与命名 exec/error exits，并保留
  value/default/blob/resource binding；新增正式 `bind-resource` authoring command，避免模板图片等资源身份丢失。
- 分叉与展开、跨多个 caller 的级联删除均作为一个 EditorSession undo/redo 单元，并在一次后端 patch 中原子提交。
- 2026-07-24 `task check` 退出码 0：contracts 一致，12 个受影响 Go 包通过，frontend format/lint/typecheck/i18n
  通过，77 个测试文件/318 项测试通过。

## References

- [Subgraph management research](../references/subgraph-management-research.md)
- [N1 definition management](stage-n1-subgraph-definition-management.md)
- [N2 interface editor](stage-n2-subgraph-interface-editor.md)
