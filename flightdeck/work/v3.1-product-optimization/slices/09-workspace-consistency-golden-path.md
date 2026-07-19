---
slice: "09"
title: 工作区一致性与黄金路径
status: completed
---

## Outcome / Question

把前八个 Slice 已恢复的工作流、目标、资源、录制、调试与计划能力串成发布前连续旅程；修正只在跨页面、重开或真实目标下出现的割裂，不再新增平行模型。

## Completion criterion

- 从新建工作流开始，可连续完成默认目标选择、宏/精准录制/视觉模板创建、资源绑定、编译、运行、调试、保存重开与计划引用。
- 工作流页、资源库、计划页和编辑器在标题、筛选、分页、空状态、选择、错误反馈与主操作层级上遵循同一产品语言。
- Windows UAC 真机目标与至少一个无需桌面目标的 portable 工作流通过；Android 入口与平台适用性不被 Windows 默认体验遮蔽。
- 数据目录不再向用户暴露测试版本语义；升级前知识和兼容说明在 3.1 发布收口时归档或清理。
- 失败点必须定位到 Source、Installation/Target、Asset、Compiler、Admission、Runtime 或 UI projection 的唯一责任层，禁止页面临时兜底制造第二事实源。

## Verification

- 使用独立验收数据创建而不是复用手工修补 fixture，保存后关闭并重开验证 durable truth。
- 执行 `task check`、production `task build`、`task webview:smoke:full` 与 UAC 真机旅程；检查关键截图、日志和 Run 时间线。
- 只在 Slice 完成后进行整阶段批量验收并提交。

## Out of scope

- 新增 OCR/AI 算法、第三方节点 UI ABI 或新的执行 runtime。
- 为未发布 3.1 保留已确认无价值的旧数据兼容分支。

## Result

Completed。最终隔离 Wails WebView 旅程从损坏 Source 隔离、新建工作流和 Run Start 开始，覆盖目录搜索/拖入、框选删除、兼容连线、保存、真实多节点 Debug Start/Step/Continue/Stop、Source-native 子图、AI review、资源工作区、离开后从 durable Source 重开，以及创建计划、引用刚创建的工作流、保存并再次打开计划。`20260719-205134` 完整旅程通过，Catalog 146 项、重开后主图 4 个节点；关键 PNG 已人工检查。

调试偶发卡住的最终根因不是 scheduler，而是 paused 事件先于控制 RPC promise 结算，验收脚本在 Step 仍 disabled 时静默点击；smoke 现在同时等待领域状态与按钮恢复 enabled，并让所有 disabled click 立即失败。Windows native smoke 同时暴露多个 Go package 并行争用全局桌面，已固定 `-p 1` 串行执行；精确/正则窗口、歧义拒绝、键盘、文本、移动、点击、拖拽、滚轮、截图、录制、codec、资产重载和 held-input 释放全部通过。

阶段门禁通过：`task check`（Go 总覆盖率 65.7%，64 个前端测试文件、250 项测试，editor gzip 212,554 / 220,000）、`task webview:smoke:full`、`task windows:smoke:automation` 和 production `task build`。最新 `bin/Yotta.exe` 为 32,237,568 bytes，manifest 固定 `requireAdministrator`，并已通过 UAC 启动。Android/Browser 与跨平台 Adapter 证据沿用 Slice 36 的 G16/G17 和 darwin/linux compile proof；稳定数据根保持 `<dataRoot>/workspace`。
