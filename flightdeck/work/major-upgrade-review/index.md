---
topic: major-upgrade-review
title: Yotta 3.1 major upgrade
summary: Implement and validate the AI-native destructive Yotta 3.1 architecture and release program.
---

## State

目标：完成并验证 AI-native、destructive 的 Yotta 3.1 架构与发布计划。

当前 Slice：ai-native-design-disposition。Go quality gate 已在 27e01b17 恢复可信绿色基线；现在逐项审计 AI-native 目标设计，只把证据明确的真实剩余 outcome 展开为后续 Slice。

## Next

落账 AI-native disposition 证据矩阵与四个精确 remaining Slices，完成独立审计提交后把当前 Slice 交接给 prompt/tool provenance boundary。

## Read now

- work/major-upgrade-review/slices/ai-native-design-disposition.md
- work/major-upgrade-review/ai-native-design.md
- knowledge/agent/codex-working-agreement.md

## Read if

- work/major-upgrade-review/slices/map.md — 选择下一 Slice、改变 blocker 或重排 frontier 时
- work/major-upgrade-review/ai-native-design.md — 执行 AI-native design disposition 或修改 AI 产品/架构边界时
- work/major-upgrade-review/slices/node-package-signing-trust.md — Go quality gate 恢复后继续 Node Package trust 时
- work/major-upgrade-review/plan.md — 调整总体阶段或最终验收边界时
- work/major-upgrade-review/design.md — 修改 3.1 总体架构边界时

## Progress

- f3c83737 完成 EditorSession shallowReactive 边界、目录单击/加号/拖放和生产 factory 回归。
- c3cab6e4 删除 composition root 全局提权，并加入 platform-scoped、production-closed 的 WebView debug options 与权限回归。
- ab5b644f 固化一键 Wails/WebView smoke；最新运行断言 99 个目录节点、画布 0→1→2、无 JS error，截图人工检查完整。
- frontend check 为 27 files / 103 tests 全绿，task build 通过；隔离 production EXE 无 UAC 冷启动并由 PrintWindow 确认首屏工作流列表正常。
- 27e01b17 以 3.1 contract/evaluator/scheduler/service/authoring、Windows adapter/input/recording、Wails composition 和 vision 算法回归恢复 Go quality gate；预算脚本 global 65.2%、根包 34.7%，所有 package floors 通过。
- c8d8b540 同工具链隔离基线为 65.3%，证明 65% 门槛有效；59.6% 漂移来自后续 destructive migration 替换高覆盖旧栈时未持续补齐新路径测试。
- unsafe.Pointer callback state 改为 CGO-independent token registry，WindowHandle 改为同构类型转换；全仓 tests、go vet、staticcheck、CGO_ENABLED=0 定向 compile 与 task check 均通过。
- AI-native 审计确认 provider-native adapters、slot/profile、OS credential、strict Extract、typed MCP 已完成；Prompt/Tool provenance、trusted instruction boundary、Agent budget、eval gate 和 AI change review/trace 是真实剩余 outcome。

## Open questions

AI-native 十项完成定义中，哪些仍属于 3.1 发布 destination、哪些已完成或因后续 provider-native/typed-patch 架构决策过时，待当前审计给出证据处置。Node Package signing trust 保持 ready frontier。