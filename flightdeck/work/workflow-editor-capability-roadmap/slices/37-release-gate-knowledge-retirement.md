---
slice: "37"
title: 3.1 发布门禁与历史 Knowledge 退役
status: pending
---

# Slice 37：3.1 发布门禁与历史 Knowledge 退役

## Outcome / Question

以 capability ledger 和 G01–G17 为唯一产品完成依据，执行一次完整 3.1 发布门禁；通过后清理或归档主动 Knowledge 中的 3.0 实现条目。

## Completion criterion

- 所有 P0/P1 ledger rows 为 verified、明确 remove，或有用户接受的删除决定。
- Windows、Android、Browser、workspace、规模、生命周期矩阵留下可复查证据。
- `task check`、production build、跨平台 compile、native smoke 和人工 UX 通过。
- 旧 Knowledge 按 retirement registry 完成 promotion、反向依赖扫描、archive/delete 和引用修复。
- `flightdeck_check` 通过；发布文档不再引用假完成或 post-3.1 延期。

## Blocked by

Slices 29–36。

## Verification

- G01–G17 适用完整矩阵。
- clean/current/corrupt workspace；1000 workflows/assets/states；target generation churn 与退出资源回收。
- 历史 Knowledge 主动搜索、依赖扫描和 exact knowledgeCount checkpoint。
- 最终用户验收后才允许重新声明 3.1 major upgrade 完成。

## Out of scope

- 代码签名、公开仓库、OSI 许可证替换等独立发布工程，除非用户另行授权。
- 不以 release notes、版本号或 production exe 存在替代能力验收。
- 不保留“发布后补 smoke”的完成状态。

## Result

Pending。
