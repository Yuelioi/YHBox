# 3.1 产品能力连续性审计与升级路线 context

## What matters

本 Work 回答 3.0 能力在 3.1 中应恢复、重做还是删除。完成证据来自 capability ledger、golden
journeys、当前源码和真实宿主门禁，不来自旧文件存在或历史截图。新产品优化已经转入
`v3.1-product-optimization`。

## Decisions

- 3.1 只保留一套 Workflow Source、Compiler、Application 和 runtime。
- 能力必须同时拥有入口、管理、创作和运行层闭环，组件存在不等于产品完成。
- Host capability、target、credential 和 policy admission 在 effect 执行前完成精确规划。
- 3.0 历史 Knowledge 只保留仍被当前源码和验收支持的结论。
- 公开签名、LICENSE 和仓库治理不作为产品能力恢复的伪开放项。

## Terms

- **Capability ledger:** 每项旧能力在 3.1 中的恢复、替代、删除和证据表。
- **Golden journey:** 跨入口、持久化、运行和用户反馈的端到端验收路径。
- **Recovery stage:** 多项相邻架构恢复工作的集中交付与验证边界。
