# Workflow 编辑器响应式与目录交互恢复

Status: completed (f3c83737)

## Outcome

Workflow 3.1 编辑器使用生产 factory 时不再因 Vue 深层 Proxy 跨 structuredClone 抛 DataCloneError；节点目录的单击、加号和拖放都能把节点加入 Source 与画布投影，并提供明确失败反馈。

## Completion criterion

- createEditorSession 使用只追踪顶层替换的 shallowReactive，不把 Source、Catalog 或 command 深代理。
- 回归通过生产 factory 执行 add-node command，同时断言 Source 和 computed 画布投影节点数各增加 1。
- 目录单击与加号复用同一添加路径；拖放只接受 Yotta 自有 MIME，并用 Vue Flow 坐标转换落点。
- 交互失败显示用户可见错误，不再表现为无响应。
- 定向与全套 frontend checks、隔离 dev WebView smoke 通过。
- 源码、测试和对应 Vue Proxy Knowledge 独立 commit。

## Blocked by

无。

## Verification

- pnpm -C frontend check：27 files / 103 tests，format/lint/typecheck/i18n/bindings/production build 全绿。
- Workflow WebView smoke：99 catalog nodes，单击 0→1、拖放 1→2，status passed。
- 20260717-011449 workflow-editor.png 已人工检查：目录、两个节点、检查器和日志 UI 完整渲染，无黑屏。
- task build 通过，editor gzip 94,712 bytes，低于 200,000 budget。

## Out of scope

- composition root 权限边界。
- dev-only CDP 与 smoke runner 的归档。
- Node Package 签名、插件 host 与最终 release gate。

## Result

完成并提交为 f3c83737 fix(frontend): restore workflow node interactions。
