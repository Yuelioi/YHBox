# 工作流创作基础与旧能力连续性 context

## What matters

恢复旧产品能力不等于恢复旧 Container runtime。每项用户能力都必须沿 3.1 Source、Contract、
Application 和 runtime 的唯一链路重新拥有入口、状态、持久化和验收。

## Decisions

- 新建工作流由后端创建最小合法 Source，前端不伪造根节点。
- Target 选择来自可信安装和 Host Profile，不允许自由文本代替 exact target。
- 资源能力通过独立资源库进入工作流，不隐藏在设置或调试入口。

## Terms

- **Capability continuity:** 用户能力在新架构每一层都有真实 owner 和可执行闭环。
- **RunStarted:** 新工作流的明确控制起点节点。
