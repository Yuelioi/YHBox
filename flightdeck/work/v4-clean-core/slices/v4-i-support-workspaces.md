# V4-I 支持工作区与规模旅程

## Goal

让全局导航、资源库和设置只暴露当前任务需要的内容，并用真实桌面旅程覆盖空数据、已有数据和
40+ Workflow，而不是用后端分页单测代替产品验收。

## Status

Completed

## Result

- 全局主导航固定为 Workflow、资源库和计划；进入编辑器后不再把“编辑 Workflow”伪装成第四个
  全局入口，Workflow 名称和保存状态由局部编辑器工具栏承担。
- 设置保持原有稳定 deep link 和组件状态，导航从八个平铺条目分成常用、连接、自动化和高级四组。
- 资源库默认是按最近使用排序的浏览态，只显示类型、搜索、创建/录制和资源内容。
- 分类、标签、排序、勾选、批量元数据和批量删除进入显式管理态；退出管理态会清除选择和高级筛选，
  回到最近使用浏览。
- 资源编辑从 `WorkflowEditorView.vue` 抽入 `EditorResourceController.ts`，以一个 command seam
  统一 Workflow Macro、InputClip 裁剪和全局 Macro 的打开、保存与错误保持。
- WebView 旅程先验证 0 个 Workflow 的启动页，再创建、保存和重开已有 Workflow，最后在隔离
  profile 创建 40 个额外 Workflow，验证总数 41、首屏 20 行和三页分页。

## Capability evidence

- 资源库浏览态和管理态均有真实截图，并完成 browse -> manage -> browse 往返。
- 批量元数据和删除的组件旅程在进入管理态后继续通过；录制、空白 Macro 和模板采集仍在浏览态。
- 设置页真实渲染四组，原 `section` query key 与 KeepAlive 组件映射未改变。
- 40+ Workflow 截图显示搜索、直接打开、行内运行和分页均可见，管理能力仍在显式入口。

## Verification

- `pnpm typecheck`
- 资源、导航、设置与 Library 相关定向测试通过。
- `go test ./cmd/workflow-editor-smoke`
- `task check`：12 个受影响 Go 包、bindings、format、lint、TypeScript、2491 个 i18n key、
  90 个测试文件 / 380 项测试通过。
- `task webview:smoke` 通过，验收目录：
  `.task/workflow-editor-smoke/20260726-213519`

## Follow-up

继续把录制生命周期和画布交互从页面协调器抽成按用户任务划分的 Module；统一所有运行入口的取消反馈，
再进入真实 `fishing-v2` 与 production profile 的最终稳定性阶段。
