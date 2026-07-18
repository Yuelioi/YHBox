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
- “用户真机测试 1”暴露的 Slices 39–42 全部完成并通过阶段真机验收。
- `task check`、production build、跨平台 compile、native smoke 和人工 UX 通过。
- 旧 Knowledge retirement、引用修复和 `flightdeck_check` 通过。
- 发布文档不再引用假完成或 post-3.1 延期。

## Blocked by

Slices 29–36、38–42。

## Verification

- G01–G17 适用矩阵。
- 简易/精准录制编辑回放、资源库保存与编辑器插入、模板选择。
- workflow default target 驱动多个节点且单节点可 override。
- 普通运行不强开面板；失败、暂停、长运行和连续单步符合 Slice 41。
- clean/current/corrupt workspace 与 `workspace-3.1 -> workspace` 迁移。
- 1000 workflows/assets/states；target generation churn 与退出资源回收。
- 最终用户验收后才允许重新声明 3.1 major upgrade 完成。

## Out of scope

- 代码签名、公开仓库、OSI 许可证替换等独立发布工程。
- 不以 release notes、版本号或 production exe 存在替代能力验收。
- 不保留“发布后补 smoke”的完成状态。

## Result

先前 Automated release candidate 判定已被 2026-07-19 用户真机验收打回；本轮修复批次已经完成，但本 Slice 继续保持 in_progress，等待用户最终接受。

- Slices 39–42 已完成：录制/资产创作、默认 target、运行工作台/调试反馈和稳定 workspace 根均有实现与回归覆盖。
- 阶段自动验收通过：task check、production build、WebView 编辑器 smoke、Windows native automation smoke 和人工截图检查均为绿色。
- 真实 bin/data 已从 workspace-3.1 原子迁移到 workspace，原工作流仍可见；当前提权版 Yotta 已启动供复测。
- 3.0 Knowledge 退役计数仍为 26；不得因本轮修复回滚已确认的知识清理。
- 只有用户完成简易/精准录制、模板确认、默认 target、Run/Step 与数据迁移的真机复测并接受，才可标记 completed 和重新声明 3.1 major upgrade 完成。
