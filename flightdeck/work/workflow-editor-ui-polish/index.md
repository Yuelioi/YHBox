# Workflow Editor UI Polish

## Goal

收口编辑器暗色图控、节点端口间距、离开确认和反馈层级回归。

## Status

Finished

## Current

四项 UI 回归已经修复并完成阶段验收：Vue Flow controls/minimap 使用 semantic dark surface，Handle
回到卡片边界，共享 ConfirmDialog 替换原生浏览器 dialog，保存与编译使用按钮原地成功反馈。

## Next

None.

## Progress

- 图控、Minimap 和节点 Handle 的暗色、位置与包围盒恢复正确。
- WorkflowEditorView 与 AI review 使用共享异步 ConfirmDialog。
- `frontend/src` 直接 alert/confirm/prompt 静态审计为零。
- 保存和编译使用短暂原地反馈，错误继续使用统一 toast。
- `task check`、WebView smoke 和人工截图检查通过。

## References

- [Frontend UI](../../knowledge/frontend/ui.md) — 当前编辑器视觉规则。
- [Headless UI verification](../../knowledge/frontend/headless-ui-verify.md) — 交互验证边界。
- [Build gates](../../knowledge/build/build.md) — 当前完整门禁。
