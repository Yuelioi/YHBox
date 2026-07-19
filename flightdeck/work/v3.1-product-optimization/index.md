---
topic: v3.1-product-optimization
title: 3.1 产品创作体验与运行工作台优化
summary: 在稳定的 3.1 架构上修正编辑器状态表达，恢复专业级多选与子图创作，重新判定调试能力，并重构复杂节点的可理解交互。
---

## State

Stage A 与 Stage B 已完成。Macro、InputClip、录制保存、工作区资源面板、运行态表达和真实多节点单步链路均已闭环；恢复批次通过完整 `task check`、production build 与增强 WebView smoke。下一实施前沿是 Slice 07 Authoring Surface。

## Next

进入 Slice 07：为复杂 typed input 建立可发现、可操作的 Authoring Surface，并以 Slice 08 的视觉分析节点作为第一条完整配方闭环。

## Read now

- work/v3.1-product-optimization/optimization-plan.md
- work/v3.1-product-optimization/slices/04-atomic-macro-model-and-editor.md
- work/v3.1-product-optimization/slices/05-precise-recording-workbench.md
- work/v3.1-product-optimization/slices/10-workspace-resource-and-runtime-recovery.md
- work/v3.1-product-optimization/slices/06-debug-runtime-workbench.md
- work/v3.1-product-optimization/context/current-vs-3.0-editor-audit.md

## Read if

- work/v3.1-product-optimization/slices/map.md — 重排阶段、选择后续 Slice 或检查完整实施前沿
- work/workflow-editor-capability-roadmap/context/user-device-test-2-design.md — 复查既有资源浏览和专业工作区设计
- knowledge/architecture/feature-continuity-across-product-stack.md — 标记录制、模板或调试能力完成前复查五层闭环
- knowledge/nodes/recording-schema-cascade.md — 修改录制 Session、pending、finalize、asset 或 playback
- knowledge/frontend/monotonic-rpc-event-snapshots.md — 修改录制或调试 RPC/事件状态合并
- knowledge/nodes/debug-step-region-runner.md — 修改 DebugController 与 scheduler 控制点
- knowledge/frontend/ui.md — 正式前端实现与视觉验收

## Progress

- Stage A 已在 9017af9f3da1ef004d732ebf28a36d4e14dc3a7f 提交，框选、状态通道和 Source-native 子图创作保留。
- Stage B 建立版本化 Macro、原子 KeyDown/KeyUp/MouseDown/MouseUp/Click/Scroll/Sleep 编辑器、安全 held-input 回放、独立精准 InputClip 工作台和两类回放节点。
- 录制 pending 丢失的根因是 Go clone 把空 `Steps`/`Tracks` 序列化为 `null`，前端严格 guard 因而丢弃整个结果；后端现在稳定输出数组，前端同时容忍旧/异常 null 集合。
- 编辑器新增键鼠宏、精准录制、视觉模板三类就地资源工作台；编辑器录制保存后插入/绑定回放节点，资源库录制仍只入库。
- 成功节点回到中性静止态，不再显示绿色边框和“已完成”；选择、当前执行、暂停、失败继续拥有各自状态通道。工具栏默认只保留运行主操作。
- 调试卡在 Run 开始的根因是 checkpoint 用新 scheduler 快照覆盖 controller generation，令 generation 从 2 回退到 1；controller 现保留并单调递增 generation。
- WebView smoke 已升级为真实多节点链：Run 开始暂停 → Step 到 Delay 暂停 → Step 完成 → 重启 → Stop，并断言编辑器资源工作台三类入口。
- `task webview:smoke`、`task check` 与 `task build` 均通过；正式 UAC `bin/Yotta.exe` 已启动并完成桌面截图检查。

## Open questions

- 无。
