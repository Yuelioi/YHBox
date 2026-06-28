# Index — target-controller-upgrade

## State

破坏性大升级 topic。调研、总体设计、Phase 1 抽象层、Phase 2 controller-call trace foundation、Phase 3 runtime trace ownership、Phase 4 keyboard controller routing、Phase 5 click controller routing、Phase 6 move controller routing、Phase 7 scroll controller routing、Phase 8 trace source metadata、Phase 9 text controller routing、Phase 10 mouse hold/drag controller routing、Phase 11 relative move controller routing、Phase 12 capture controller routing、Phase 13 action trace events、Phase 14 frontend trace log consumer、Phase 15 action log polish、Phase 16 action trace drawer、Phase 17 redacted trace persistence、Phase 18 backend capability profiles、Phase 19 Android ADB controller、Phase 20 Browser CDP controller、Phase 21 runtime active target 已完成并提交。核心决策：Go 保持主运行时，Rust 只作为 Win32/native controller hot path；先引入 `Target / Controller / CoordinateSpace / Trace`，再迁移节点、Android、浏览器和输入后端矩阵。

## Next

Plan next slice: add runtime controller factory for active target, then add target-selection/discovery nodes for Android/Browser.

## Read now

- design.md
- plan.md
- plans/phase2-trace.md
- plans/phase3-runtime-trace.md
- plans/phase4-keyboard-controller.md
- plans/phase5-click-controller.md
- plans/phase6-move-controller.md
- plans/phase7-scroll-controller.md
- plans/phase8-trace-source.md
- plans/phase9-type-text-controller.md
- plans/phase10-mouse-hold-drag-controller.md
- plans/phase11-relative-move-controller.md
- plans/phase12-capture-controller.md
- plans/phase13-action-trace-events.md
- plans/phase14-frontend-action-trace-log.md
- plans/phase15-action-log-polish.md
- plans/phase16-action-trace-drawer.md
- plans/phase17-action-trace-file-persistence.md
- plans/phase18-controller-backend-profiles.md
- plans/phase19-android-adb-controller.md
- plans/phase20-browser-cdp-controller.md
- plans/phase21-runtime-active-target.md
- ../../knowledge/architecture/target-controller-phase3-notes.md
- ../../knowledge/architecture/target-controller-phase4-notes.md
- ../../knowledge/architecture/target-controller-phase5-notes.md
- ../../knowledge/architecture/target-controller-phase6-notes.md
- ../../knowledge/architecture/target-controller-phase7-notes.md
- ../../knowledge/architecture/target-controller-phase8-notes.md
- ../../knowledge/architecture/target-controller-phase9-notes.md
- ../../knowledge/architecture/target-controller-phase10-notes.md
- ../../knowledge/architecture/target-controller-phase11-notes.md
- ../../knowledge/architecture/target-controller-phase12-notes.md
- ../../knowledge/architecture/target-controller-phase13-notes.md
- ../../knowledge/architecture/target-controller-phase14-notes.md
- ../../knowledge/architecture/target-controller-phase15-notes.md
- ../../knowledge/architecture/target-controller-phase16-notes.md
- ../../knowledge/architecture/target-controller-phase17-notes.md
- ../../knowledge/architecture/target-controller-phase18-notes.md
- ../../knowledge/architecture/target-controller-phase19-notes.md
- ../../knowledge/architecture/target-controller-phase20-notes.md
- ../../knowledge/architecture/target-controller-phase21-notes.md

## Read if

- ../../knowledge/architecture/automation-framework-survey.md — 需要回看 ok-script / MaaFramework / Airtest / RPA 调研结论。
- ../../knowledge/architecture/target-controller-upgrade-guide.md — 需要回看长期升级路线、Go/Rust 分工、Android/Win32/Browser 策略。
- ../../knowledge/nodes/node-system-architecture.md — 迁移节点或 runtime service 前。
- ../../knowledge/subgraph/asset-subsystem.md — 改截图取点、资产 capture、模板变体前。
- ../../knowledge/input/sendinput-primitive-size-and-return.md — 调 Win32 SendInput primitive 前。

## Progress

Done:
- 市面框架调研。
- 总体设计 spec。
- Phase 1 implementation plan。
- cockpit 中加入恢复入口。
- Phase 1 代码：`internal/automation/target`、`internal/automation/controller`、runtime WindowHandle -> Target bridge。
- Phase 2 代码：`internal/automation/trace`、Win32Controller 可选 controller-call trace hook。
- Phase 3 代码：`RuntimeContext` 拥有 per-run trace recorder，并提供 `TraceRecorder` / `TraceRecords` / `ClearTrace`。
- Phase 4 代码：`InputService.KeyDown/KeyUp` 经 `Win32Controller` 执行，并写入 runtime trace。
- Phase 5 代码：`InputService.Click` 经 `Win32Controller` 执行，并写入 runtime trace。
- Phase 6 代码：`InputService.MoveTo` 经 `Win32Controller` 执行，并记录最小 coordinate step。
- Phase 7 代码：`InputService.Scroll` 经 `Win32Controller` 执行，并记录最小 coordinate step。
- Phase 8 代码：controller action trace 增加 `ActionSource`，framework dispatch 的输入动作带 container/node/kind/in-pin 来源。
- Phase 9 代码：`InputService.TypeText` 经 `Win32Controller.Text` 执行，并写入带 source 的 `text` trace。
- Phase 10 代码：`InputService.MouseDown/MouseUp/Drag` 经 `Win32Controller` 执行，并写入带 source 的 mouse/drag trace。
- Phase 11 代码：`InputService.MouseMoveRel` 经 `Win32Controller.MoveRelative` 执行，并写入带 source 的 `move-relative` trace。
- Phase 12 代码：`CaptureService.Capture/CaptureROI` 经 `Win32Controller.Screenshot` 抓全帧，并写入带 source 的 `screenshot` trace。
- Phase 13 代码：controller action trace 通过 `container:action-trace` runtime event 导出，同时保留 memory trace。
- Phase 14 代码：前端订阅 `container:action-trace`，写入 log store 的结构化 `actionTraces` 和 `action` 日志行。
- Phase 15 代码：LogPanel 为 `action` 日志级别添加专门颜色。
- Phase 16 代码：LogPanel 增加动作 Trace 查看器，按结构化 `actionTraces` 展示 action/status/source/target/backend/duration/payload。
- Phase 17 代码：`container:action-trace` 通过现有 `LogSink` 写入脱敏 JSONL 文件行，不落 raw request/result/句柄。
- Phase 18 代码：`internal/automation/controller` 增加 Win32/Android ADB/Browser CDP/mock/replay 后端能力 profiles。
- Phase 19 代码：增加可测试的 `AndroidADBController`，覆盖 screencap/tap/swipe/text/start/stop，并写入 action trace。
- Phase 20 代码：增加可测试的 `BrowserCDPController`，覆盖截图、鼠标、滚轮、键盘、文本 CDP 调用，并写入 action trace。
- Phase 21 代码：`RuntimeContext` 增加 active `target.Target`，`SetActiveWindow` 同步 Win32 target，input/capture adapters 通过 active target 构造 controller。

Current:
- 下一刀：active target controller factory，然后做 Android/Browser target-selection/discovery 节点。

## Open questions

- Phase 3 不改变节点路由；真正把节点动作通过 Win32Controller 执行要另写 Phase 4 plan。
- Phase 8 source metadata 覆盖 framework dispatch 的 input action；直接 `NewInputAdapter(rt)` 调用仍保持空 source。
- Capture node 已经进入 controller trace；Vision adapter/cached frame 仍直接走 runtime capture backend。
- `container:action-trace` 已可订阅、前端查看、脱敏写文件；尚未实现历史文件浏览 UI、批处理策略、raw payload opt-in。
- Backend profiles、Android ADB controller skeleton、Browser CDP controller skeleton、runtime active target 已落代码；尚未把 non-Win32 controllers 接入 runtime factory，也尚未做 CDP/ADB discovery。
