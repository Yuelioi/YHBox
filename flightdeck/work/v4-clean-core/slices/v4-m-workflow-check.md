# V4-M 工作流检查反馈

## Goal

让用户明确知道该动作是在检查当前工作流，而不是生成二进制；无论检查通过或发现问题，点击后都必须
有可见反馈，并且检查不能偷偷保存编辑中的草稿。

## Status

Finished

## Delivered

- 产品按钮、工具命令、问题面板和中英文提示统一使用“检查工作流 / 工作流问题”。
- Wails Workflow service 新增 `CheckDraft`，直接检查编辑器当前内存 Source；显式检查不调用保存。
- 可达节点缺少必填输入仍是 error；断开执行路径的草稿节点改为 warning，并继续从 Program 中排除。
- 没有问题时显示成功 toast；有问题时自动打开问题面板并保留节点定位能力。
- WebView smoke 实际添加未完成节点、点击检查、确认 `MISSING_INPUT_BINDING` 可见，并确认草稿仍未保存。

## Decisions

- Source 与 Program 仍是不同语义的 canonical JSON；内部 `Compiler` / `CompileDraft` 名称保留。
- `CompileSource` 保留给现有 CLI、MCP 和内部调用；编辑器产品路径使用 `CheckDraft`。
- warning 不参与 `schema.HasErrors`，因此断开草稿不会阻断当前可达运行路径。

## Verification

- `task check`
- `task webview:smoke`
- `go test ./internal/workflow/compiler ./internal/services/workflow -count=1`
- `pnpm exec vitest run src/app/editor/EditorRunController.test.ts src/app/editor/editorToolbarModel.test.ts src/app/editor/EditorSession.test.ts`
