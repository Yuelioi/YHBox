# Cockpit — YHFish

**Last updated**: 2026-06-08 by 月离 (颜色范围吸管 ship 完成：schema widget 标记 + ExtractColorRange 纯 CV + picker color 模式 + StructuredInput 吸管按钮 + i18n，真机 smoke 过。spec/plan done+归档。)
**Active focus**: 资产子系统重构 — 模板/clip 改 Unity 式 GUID + 全局内容寻址 blob 池。spec 已落 (graduate)：[asset-subsystem-guid-cas](specs/2026-06-09-asset-subsystem-guid-cas.md)。待用户复审 spec → 写实现 plan。

## 进行中

(无)

## 下一步

用户复审 spec [asset-subsystem-guid-cas](specs/2026-06-09-asset-subsystem-guid-cas.md) → 拍板后写实现 plan（flightdeck plan，按 spec §10 改动面分相位：新 asset 包 → 运行时匹配 → 节点/校验/依赖 → 分享导入坍缩 → 前端 picker/捕获 → 接线）。决策已定：方案 B（GUID+CAS）、全局 blob 池、模板+clip 一起、资产独立于图存在、零迁移。

## Hanging tasks

- [ ] 无阻塞待办。（原积压已路由：编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md)；bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md)；删符号全仓 grep → [checklists/code-style.md](checklists/code-style.md)；i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)；诊断探针目录已删。）
