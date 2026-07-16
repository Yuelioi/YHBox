---
topic: major-upgrade-review
title: Yotta 3.1 major upgrade
summary: Implement and validate the AI-native destructive Yotta 3.1 architecture and release program.
---

## State

目标：完成并验证 AI-native、destructive 的 Yotta 3.1 架构与发布计划。

当前 Slice：restore-go-quality-gate。Workflow 编辑器交互、桌面普通权限启动和 WebView 自调试已分别验证并提交；最终交付检查发现全仓 Go quality gate 在本批次之前已经漂移为红，必须先恢复可信门禁，再继续 Wave E。

## Next

用已建立的 coverage、go vet 和 staticcheck 红灯逐项定位并恢复 Go quality gate；不降低阈值或绕过检查，最终以完整 task check 全绿和独立 commit 收口。

## Read now

- work/major-upgrade-review/slices/restore-go-quality-gate.md
- knowledge/agent/codex-working-agreement.md
- knowledge/build/build.md

## Read if

- work/major-upgrade-review/slices/map.md — 选择下一 Slice、改变 blocker 或重排 frontier 时
- work/major-upgrade-review/slices/ai-native-design-disposition.md — Go quality gate 恢复后审计 AI-native 设计完成度时
- work/major-upgrade-review/ai-native-design.md — 执行 AI-native design disposition 或修改 AI 产品/架构边界时
- work/major-upgrade-review/slices/node-package-signing-trust.md — Go quality gate 恢复后继续 Node Package trust 时
- work/major-upgrade-review/plan.md — 调整总体阶段或最终验收边界时
- work/major-upgrade-review/design.md — 修改 3.1 总体架构边界时

## Progress

- f3c83737 完成 EditorSession shallowReactive 边界、目录单击/加号/拖放和生产 factory 回归。
- c3cab6e4 删除 composition root 全局提权，并加入 platform-scoped、production-closed 的 WebView debug options 与权限回归。
- ab5b644f 固化一键 Wails/WebView smoke；最新运行断言 99 个目录节点、画布 0→1→2、无 JS error，截图人工检查完整。
- frontend check 为 27 files / 103 tests 全绿，task build 通过；隔离 production EXE 无 UAC 冷启动并由 PrintWindow 确认首屏工作流列表正常。
- task check 在 Go coverage 59.6% < 65% 停止；detached 53e6d8a9 同工具链为 59.8%，另有既存 go vet unsafe.Pointer 与 staticcheck S1016，均转入当前独立 Slice。

## Open questions

coverage 65% 与实际 detached HEAD 59.8% 的漂移源尚待定位；修复必须区分真实测试债、coverage scope 错误和预算契约漂移，不能先假定答案。Go gate 恢复后，AI-native design disposition 与 Node Package signing trust 都是可选 ready frontier。
