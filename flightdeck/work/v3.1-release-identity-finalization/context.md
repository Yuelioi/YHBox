# Yotta v3.1 Release Identity Finalization context

## What matters

Release version 是制品身份，不是代码架构命名。工程候选已固定为 3.1.0，但当前 LICENSE 仍是
source-available，且未具备公开 stable 所需签名与治理条件。

## Decisions

- 版本号只进入正式元数据、manifest、artifact 和 tag。
- 公开 stable 前置由独立 release/governance Work 持有，不污染产品实现。

## Terms

- **Release identity:** 用户和发布系统识别制品版本的权威元数据集合。
- **Engineering candidate:** 通过仓库工程门禁但尚未满足公开发布治理的候选制品。
