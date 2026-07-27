# V4-S Go 生产代码瘦身

## Goal

在不削弱现有功能、持久化兼容和运行安全的前提下，删除结构清扫中新增的浅 Module、重复决策与过度防御，
使相对 `2d2c2226` 的生产 Go 代码从净增转为净减。

## Status

Finished

## Baseline

- 生产 Go：新增 2648 行、删除 2402 行，净增 246 行。
- Go 测试：新增 702 行、删除 129 行，净增 573 行；测试单独记账，不用于掩盖生产代码增长。
- WebView smoke 工具：新增 2869 行、删除 2885 行，净减 16 行。
- Go 文件：659 个增至 675 个；拆文件本身不作为轻量化证据。

生产 Go 统计排除 `*_test.go` 与 `cmd/workflow-editor-smoke/`。不以压缩排版、删除有价值注释或合并
无关职责制造行数下降。

## Current

最终权威统计为生产 Go 新增 2546 行、删除 2617 行，净减 71 行；Go 测试净增 599 行、WebView
smoke 工具净增 64 行，继续单独记账。已删除 `sourceLibrary` 双层对象与五个 facade 转发、重复的
Source patch candidate、coordinator 第二套 lifecycle/job 状态、Application 第二把 lifecycle lock、
`Runtime.Start` 与 environment retire 等一次性转发。第二轮进一步把 automation generation 所有权
收回唯一 appbootstrap Runtime，删除 localruntime 的重复 BlobStore 投影、provider digest/merge
小对象，以及 Application 的 command → private implementation 转发。

Run coordination、Source transition、admission、local runtime、Adapter ABI 与 generation lease
保留为承担独立不变量的 seam。首次最终 `task check` 捕获 MCP 测试残留的已删除 Runtime.Start 调用；
修正并定向验证后，最终门禁重试通过。

## Next

None.

## Acceptance

- 相对 `2d2c2226` 的生产 Go 净变化小于 0，且每批变更都说明删除了哪项复杂度。
- 保留唯一 production Executor、admission-before-authority、runtime 生命周期与 durable migration。
- 现有产品行为、Wails contract、`fishing-v2` 和 WebView 主旅程不退化。
- 中途不运行 `task check`、全量 build 或完整 smoke；整个瘦身完成点只统一验收一次。

## Verification

- `task check`：通过；32 个受影响 Go 包、93 个前端测试文件 / 394 项测试、bindings、插件、
  AI eval、格式、类型和 i18n 全绿。
- `task build`：通过；正式 Windows GUI、CLI、ScriptWorker、Wasm runner、版本/manifest、
  bundle budget 与隔离 desktop startup smoke 全绿。
- `task webview:smoke:fishing`：通过；10 个画布节点、6 个子图调用，首屏 2437ms、启动 5217ms，
  目录 `.task/workflow-editor-smoke/20260728-021907`。
- `task webview:smoke:full`：通过；5 个工作区工具及编辑器、子图、资源、计划、启动器旅程全绿，
  启动 3213ms，目录 `.task/workflow-editor-smoke/20260728-022022`。
