# Asset Workbench Upgrade

## Goal

把模板、蓝图、录制与 Clip 资源升级为可搜索、可管理、可预览的商业级工作台。

## Status

Finished

## Current

历史实现范围已经完成：资产卡片、详情、分辨率变体、搜索筛选、显示偏好和编辑器能力边界均已
落地并通过当时的 `task check` 与 production build。旧记录中等待的桌面视觉反馈不再保持为
活动任务；若新反馈仍可复现，应基于当前资源工作区新开 Work。

## Next

None.

## Progress

- 建立模板、蓝图和 Clip 的统一浏览、详情、筛选与管理界面。
- 将分辨率变体动作收敛为内容宽度，并同步普通与展开详情。
- 移除会隐藏真实能力的“基础/专业”模式，改为仅控制技术信息密度的显示偏好。
- 变量、Snippets、调试、节点改名、Expr 修复和输出绑定始终保持可用。
- 当时完整 `task check`、前端 613 项测试与 `task build` 通过。

## References

- [Asset subsystem](../../knowledge/subgraph/asset-subsystem.md) — 当前资源边界。
- [Frontend UI](../../knowledge/frontend/ui.md) — 产品视觉与交互约定。
- [Build gates](../../knowledge/build/build.md) — 当前验证入口。
