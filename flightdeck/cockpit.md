# Cockpit — YHFish

**Last updated**: 2026-06-08 by 月离 (颜色范围吸管 ship 完成：schema widget 标记 + ExtractColorRange 纯 CV + picker color 模式 + StructuredInput 吸管按钮 + i18n，真机 smoke 过。spec/plan done+归档。)
**Active focus**: 资产子系统重构实现中。spec(graduate)+plan 已落并 commit。subagent 执行：**Phase 0-1 完成**(asset 包 blob 池+记录库+PickVariant+GC，-race 全绿，已 commit)。下一步 Phase 2 起的耦合改造(matcher/节点/clip/分享/RPC/前端/接线)。

## 进行中

(无)

## 下一步

按 [plan](plans/2026-06-09-asset-subsystem-guid-cas.md) 续执行 **Phase 2**：运行时匹配 `templateMatcherAdapter` 改全局 asset store + Detect 按 GUID + 缓存按 blob sha（`wire_container.go`）。注意 Phase 2-8 是耦合 swap，`go build ./...` 会红到整批完成才绿；**Phase 8 真机 smoke 必须用户在自己机器跑**（真实游戏窗口+模板）。已完成 commit：b90b584(P0)→c900d36(P1)。

## Hanging tasks

- [ ] 无阻塞待办。（原积压已路由：编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md)；bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md)；删符号全仓 grep → [checklists/code-style.md](checklists/code-style.md)；i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)；诊断探针目录已删。）
