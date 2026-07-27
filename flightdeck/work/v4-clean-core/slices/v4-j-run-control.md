# V4-J 一致运行控制

## Goal

让用户从不同入口启动 Workflow 后都能看懂当前状态，并在入口拥有确切 Run 身份时直接停止；不为
Schedule 另造一套平行运行协议。

## Status

Completed

## Result

- Workflow 列表不再永久停在“已开始”；它记录 Workflow 对应的当前 Run，消费 Run 事件并短轮询
  极短 Run，显示运行中、完成、停止或失败。
- 列表行在 Run 活跃时将运行按钮原位替换为停止按钮，只取消该行对应的确切 Run。
- 悬浮启动器的运行项可再次点击停止，停止是独立的中性结果，不再显示成启动失败。
- 启动器完成、失败和停止反馈保持 10 秒，独立窗口或高负载下仍能被用户看见。
- 全局状态栏从“最后一条 Run 事件”改为活跃 Run 集合；多个 Workflow 并发时不会因其中一个结束
  就误显示就绪。“全部停止”继续覆盖从 Schedule 启动且主页面没有单独 Run ID 的运行。
- 编辑器仍通过 `EditorRunController` 停止当前 Run；Schedule 继续只负责触发，不持有第二套运行状态。

## Capability evidence

- 组件旅程真实验证列表运行按钮变为停止按钮，点击后使用同一个 Run ID 并恢复运行按钮。
- 启动器 full WebView 旅程通过独立 WebView 启动 Workflow，并捕获明确成功状态。
- 全局状态栏源契约验证活跃集合与统一 `cancelAllRuns`。

## Verification

- `pnpm typecheck`
- 运行控制相关 31 项定向测试通过。
- `task check`：12 个受影响 Go 包、bindings、format、lint、TypeScript、2491 个 i18n key、
  90 个测试文件 / 380 项测试通过。
- `task webview:smoke:full` 通过，验收目录：
  `.task/workflow-editor-smoke/20260726-215114`

## Follow-up

下一阶段只处理尚未完成的录制/画布 Module、`fishing-v2` 自动保留基线和性能预算，不扩展新的运行身份
或 Schedule 状态模型。
