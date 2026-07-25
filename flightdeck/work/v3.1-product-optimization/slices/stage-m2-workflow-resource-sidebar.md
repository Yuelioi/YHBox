# M2a — 工作流资源侧栏与本机素材快照

## Journey

工作流编辑器左侧的 Macro、精准录制与视觉模板面板原先只查询本机素材库，无法区分“当前工作流拥有的
可移植资源”和“本机可复用素材”；列表也只有使用入口，缺少标签筛选、数字分页、删除和批量管理。

本 Slice 建立第一个真实 M2 纵向切片：

- 资源种类继续由左侧一级工具选择；面板内部只切换“当前工作流 / 本机素材库”。
- 单一列表持续突出录制、截图、绑定和拖放；复选框、overflow 菜单与选中后出现的紧凑工具条承载
  逐项选择、本页全选、元数据编辑和批量删除，不再维护“使用 / 管理”两套视图。
- 从本机素材库使用素材时读取完整图片/Macro/InputClip metadata，向 Source 写入独立 Workflow Resource
  snapshot，再以 Resource Binding 绑定节点；后续全局元数据修改或删除不改变该 Source snapshot。

## Implementation

- Workflow authoring patch 新增 `add-resource`、`update-resource-metadata`、`remove-resource`，复用精确 revision、
  canonical parse 和原子 patch；删除扫描全部 Node 与 GraphCall binding，被引用资源返回 `RESOURCE_IN_USE`。
- `EditorSession` 支持同构命令和 `batch` 复合编辑；批量更新/删除以及“新增 snapshot + 绑定/建节点”各占一个
  undo 单元。
- `WorkflowResourceDock` 统一搜索、分类、标签、排序、20/50/100 page size 和数字分页；选择跨页保留，
  本页全选只影响本页，分类与排序在窄侧栏内等宽铺满。
- 当前工作流资源完全从 `source.resources` 和 graph bindings 派生，不新增前端资源 store；本机素材继续使用
  Asset Service 的 server-side query、metadata 和 batch API。
- 本机图片读取全部 resolution/bbox/blob 变体；Macro 固化 base resolution、action count、duration 和 blob；
  InputClip 固化 duration、event count、recording/mouse mode、base resolution、counts/360、stop hotkey 和 blob。
- 共享 `BlobPreview` 为资源库、选择器和节点绑定提供大图预览、适应窗口、实际尺寸与 25%–400% 缩放。
  节点和 GraphCall 的通用资源字段把名称区域作为定位入口：Resource Binding 使用
  `resourceId + variantId` 定位当前工作流，旧 Blob Binding 通过 Asset Service 解析稳定 GUID 后定位本机素材库；
  左侧自动切换资源种类/范围、以精确 ID 过滤并聚焦高亮唯一行。

## Verification

- Go authoring lifecycle 覆盖新增、规范化元数据、删除和引用保护；生成合同的精确 tagged union 已更新。
- `workflowResourceLibrary.test.ts` 使用 1000 个资源验证搜索、分类、标签、倒序、数字分页和 Node/GraphCall
  引用计数。
- `EditorSession.test.ts` 验证资源 batch 只占一个 undo，并持久化为三个正式 patch command。
- `task check` 退出码 0：13 个受影响 Go 包、前端 format/lint/typecheck/i18n、80 个测试文件/343 项测试通过。
- `task webview:smoke` 最终退出码 0；修正烟测把 L 形布局第三节点框入、以及只传 mouse modifier 导致
  Ctrl 多选始终为 1 的两个确定性问题。最终 `20260725-112124/resource-tools.png` 已目检，320px 侧栏无重叠。
- 资源侧栏改为 editor 内按需加载，并用 foundations 回归测试锁定异步边界；production editor 初始 gzip
  从 223692 降至 201936 bytes，`task build` 通过 bundle gate、Windows GUI metadata 和 5 秒隔离启动 smoke。
- 真机反馈后删除“使用/管理”双模式；WebView smoke 同时断言 scope 只有一个激活项、激活/非激活
  computed style 不同、模式按钮为 0、两列筛选宽差不超过 2px 且填满容器。烟测还从实际 listener 选择
  IPv4 或 IPv6 CDP endpoint，避免 WebView2 监听 `::1` 时误报首轮启动失败。
- 资源预览/定位增量验证通过：`resourceLocator.test.ts` 锁定精确 variant 与三类面板映射，Asset Service
  在 1000 条 fixture 中用 GUID 只返回唯一资源；最终 `task check` 覆盖 29 个 Go 包与 81 个前端测试文件/
  347 项测试，production build（editor 201985/220000 gzip）和同一执行单元持续等待的 Windows WebView
  smoke 均退出 0。门禁还补齐 M2 authoring resource 模型的 Wails RPC 快照（202 → 211 models）。

## Remaining M2

- 编辑器内新录制/新截图仍沿用现有 Global Asset 保存完成链，需要把 finalize target 改成 Workflow Resource；
  资产库创建继续保持 Global Asset。
- Workflow Resource 显式提升为 Global Asset、duplicate/shared update 与删除原 Global Asset 后的完整导出运行验收
  尚未实现。
- 本机素材拖放目前仍保留旧 BlobRef 快捷路径；按钮/双击已走 snapshot，拖放应在下一 Slice 收敛到同一语义。
