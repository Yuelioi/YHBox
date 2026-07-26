# M2b — 编辑器内 Workflow Resource 创作

## Journey

编辑器内的新录制和新截图当前先创建 Global Asset，再把其 BlobRef 或快照放进工作流；本机素材拖放也仍走
旧 Blob Binding 快捷路径。这让“在哪里创建”无法决定资源归属，并使工作流资源语义在按钮、双击与拖放之间
不一致。

本 Slice 建立单一创作语义：

- 资产库内录制/截图继续创建 Global Asset。
- 工作流编辑器内录制/截图直接写共享 CAS，并返回完整 Workflow Resource；不创建临时 Global Asset。
- 本机素材按钮、双击和拖放都先创建完整 snapshot，再通过 Resource Binding 使用。
- Source 的资源新增和节点绑定继续占一个 undo 单元；未被 Source 引用的中断产物由现有 CAS 宽限 GC 回收。

## Implementation

- 新建 Workflow Resource authoring Module；其 Interface 接受图片、Macro 或 InputClip 内容并返回规范化
  `schema.WorkflowResource`，Blob/CAS 编码、校验、ID 分配和 metadata 规范化留在 Implementation 内。
- Recording finalize 使用显式 destination 区分 `global-asset` 与 `workflow-resource`；结果是带 discriminator
  的精确 union，编辑器不再从 Global Asset 二次读取或拼装资源。
- Screen Picker 增加 workflow-resource capture mode；Global Asset 保存模式保持原行为。
- Global Asset → Workflow Resource 的 snapshot 投影抽成共享前端 Module，资源侧栏使用与拖放复用同一入口。

## Verification

- Pending recording 已验证两个 destination：Global Asset 写 Catalog；Workflow Resource 只写 CAS，并返回
  完整 Macro/InputClip metadata。图片创作覆盖 data URL、resolution/bbox、metadata 与 BlobRef。
- 前端 82 个测试文件/351 项测试通过；精确 finalize union、截图资源事件、三类 Global Asset snapshot 和
  GUID-only 不可信拖放 payload 均有定向覆盖。
- `task check` 以同一执行单元退出 0：router self-test、AI eval 8/8、Wails 17 services/143 methods/
  213 contract models、34 个受影响 Go 包、frontend format/lint/typecheck/i18n 全部通过。
- `task build` 退出 0：editor 202909/220000 gzip，Windows GUI metadata 正确，隔离 RootSet 启动存活 5 秒。

## Status

Finished.
