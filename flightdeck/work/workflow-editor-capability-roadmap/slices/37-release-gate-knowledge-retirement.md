---
slice: "37"
title: 3.1 发布门禁与历史 Knowledge 退役
status: in_progress
---

# Slice 37：3.1 发布门禁与历史 Knowledge 退役

## Outcome / Question

以 capability ledger、G01–G17 和用户真机旅程为唯一产品完成依据；自动矩阵不能替代真实创作、运行与调试闭环。

## Completion criterion

- 所有 P0/P1 ledger rows 为 verified、明确 remove，或有用户接受的删除决定。
- Windows、Android、Browser、workspace、规模、生命周期矩阵留下证据。
- 用户真机测试暴露的所有重新打开 Slice 均完成并通过阶段真机验收。
- `task check`、production build、跨平台 compile、native smoke 和人工 UX 通过。
- 旧 Knowledge retirement、引用修复和 `flightdeck_check` 通过。
- 发布文档不再引用假完成或 post-3.1 延期。

## Blocked by

Slices 29–36、38–42；其中 Slice 39 与 Slice 41 的实现和自动门禁已完成，但仍等待用户真机接受。

## Verification

- G01–G17 适用矩阵。
- 简易/精准录制编辑回放、资源库保存与编辑器插入、模板整卡选择和键盘确认。
- workflow default target 驱动多个节点且单节点可 override。
- Debug Start 必须稳定落在 paused；连续 Step、Continue、Stop 状态准确；终态不再显示“即将执行”。
- 500+ 资产可通过高密度列表、搜索、筛选、排序、分页和批量动作管理；切换视觉模板后不显示录制控制。
- picker 与 Inspector 的资源选择必须整项可选、选择状态清晰、资源引用为单一复合字段。
- clean/current/corrupt workspace 与 legacy 迁移。
- 最终用户验收后才允许重新声明 3.1 major upgrade 完成。

## Out of scope

- 代码签名、公开仓库、OSI 许可证替换等独立发布工程。
- 不以 release notes、版本号或 production exe 存在替代能力验收。
- 不保留“发布后补 smoke”的完成状态。

## Result

In progress。2026-07-19 用户真机测试 2 的整改阶段已完成并通过 `task check`、production build、WebView smoke 和人工截图检查；Slice 39/41 仍等待用户使用 UAC production build 与真实 workspace 完成接受。核心 Source/Compiler/Adapter/Asset Store/Run Journal 架构继续保留，Slice 37 在接受完成前不解除发布阻断。
