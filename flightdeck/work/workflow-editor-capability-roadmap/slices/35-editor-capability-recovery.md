---
slice: "35"
title: 编辑器与节点能力连续性恢复
status: completed
---

# Slice 35：编辑器与节点能力连续性恢复

## Outcome / Question

恢复新用户可发现的完整创作路径，并在 3.1 强类型/安全模型内补齐窗口、held input、Stopwatch、观察、Switch、图像和状态便利能力。

## Completion criterion

- Node/State/Asset/Target picker 使用一致搜索与上下文候选交互。
- 新建 Start、Delete、快捷键、布局、连线、端口提示、debug 形成 G01 完整旅程。
- 恢复 P0/P1 节点：window lifecycle、held input、Stopwatch、WaitStable/WaitChange、typed Switch、image I/O、状态便利操作。
- Repeat.index 等 typed output 可直接发现合法消费者和转换。
- specialized vision/EventTick 通过旧 workflow/真实任务证据作最终恢复、复合替代或删除决定。

## Blocked by

Slices 33–34。

## Verification

- G01、G06、G09–G12；较大 catalog、1000 assets、large Run State fixture。
- projection/compiler connection-plan parity 和跨图 type-change impact tests。
- 阶段末统一 editor/integration gate 与人工 UX 验收，再形成 Stage R3 commit。

## Out of scope

- 不恢复 ambient Expr、yt console、旧 Pin DTO 或第二 runtime。
- 不按缺一个节点就增加前端名称白名单。
- 不以节点数量替代能力闭环。

## Result

Completed。

- State 面板支持搜索、100 项渐进显示、拖拽/直接插入 Read/Write，以及 LastChange/原子 Increment 上下文动作；1000 states fixture 通过。
- Target/State config 使用可搜索、可虚拟化 SelectMenu；Node catalog/connection candidates 与共享 Asset Picker 保持搜索式交互。
- 恢复 typed Switch、显式数据流 Stopwatch Start/Read/Stop、WaitStable/WaitChange、workspace-safe Load/Save Image；窗口生命周期和 held input 沿用 R2 已验证实现。
- WaitStable exact-target capture/journal 和 PNG workspace → BlobRef → workspace round-trip 通过真实 executor/admission/provider；workspace 覆盖写使用 staged durable replace，非覆盖写原子拒绝 clobber。
- EventTick 删除为 ambient background subrunner；Run 内用 Repeat/Wait/Delay，独立周期用 Schedule interval。specialized vision 由 Capture + FindColorBlobs/AnalyzeColor + typed list/geometry/math 复合替代，旧名称均可在目录搜索到。MouseCalibration 保留为机器 input profile/HUD，不进入 Workflow Source。
- 1000 workflows 查询分页、1000×2 assets、1000 states、145 nodes 与 Repeat.index/connection-plan/compiler parity 均有自动化证据。
- 阶段批量门禁：`task check` 全绿（Go、vet/staticcheck、coverage、44 个前端文件 185 tests、production frontend/bundle）；Wails WebView G01/G09/G11/G12 smoke 通过并人工检查三张 PNG，运行目录 `.task/workflow-editor-smoke/20260718-223440/`。

