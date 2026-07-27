# V4-H 聚焦编辑器

## Goal

让编辑器默认只突出画布、保存和运行，同时保留子图、资源、状态、AI、调试和运行检查能力；将命令层级
从页面模板中收进可测试的深 Module。

## Status

Completed

## Result

- 顶栏默认只保留返回、名称与保存状态、撤销/重做/定位、运行、保存和“工具”入口。
- AI、Run 状态、属性、编译、诊断、时间线、启动调试、Workflow 设置与刷新进入同一个按需工具菜单。
- 调试器只在调试上下文激活后常驻；录制控制只在录制生命周期中出现。
- 左侧工作区从五个常驻入口收敛为“子图”和“资源工具”；Macro、InputClip、视觉模板和 Snippet
  仍为独立资源面板。
- 默认 Target 从常驻大选择器改为图上下文中的紧凑弹出选择器，仍只保存 Target slot，不写入用户
  应用路径或设备配置。
- `editorToolbarModel.ts` 以一个 context 生成全部命令层级；Toolbar 只发出一个语义 command。
- `EditorRunController.ts` 以 `execute(command)` 统一编译、保存、运行、调试、取消、刷新和时间线翻页，
  页面不再重复承担反馈与工作台路由。
- 后续 `EditorResourceController.ts` 以相同 command seam 接管 Workflow Macro、InputClip 裁剪和全局
  Macro 编辑，失败时保留编辑态。

## Capability evidence

- 子图管理、Macro、InputClip、视觉模板、Snippet、状态、AI、诊断、时间线和调试入口均由组件测试或
  Windows WebView 旅程覆盖。
- WebView smoke 实际打开折叠菜单内的状态、AI、调试、三类资源，完成子图、保存、运行与计划旅程。
- 编辑器默认左侧入口数从 5 降到 2；最终截图中底部 Runtime Workbench 只在需要时展开。

## Verification

- `pnpm typecheck`
- `task check`：12 个受影响 Go 包、bindings、format、lint、TypeScript、2478 个 i18n key、
  87 个测试文件 / 371 项测试通过。
- `go test ./cmd/workflow-editor-smoke`
- `task webview:smoke` 通过，验收目录：
  `.task/workflow-editor-smoke/20260726-210036`

## Follow-up

`WorkflowEditorView.vue` 仍有 4,000 余行。下一步继续按用户任务抽出资源/录制工作台和画布交互 Module；
不做机械式 composable 拆分。
