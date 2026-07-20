# IO / JSON / Fetch Nodes Plan context

## What matters

这是没有最终实现证据的历史计划。HTTP 与文件 IO 在 3.1 中必须通过精确 Capability Requirement、
admission 和可信 host provider，不得按旧方案直接获得环境权限。

## Decisions

- 旧 checklist 只提供需求线索，不定义当前 Node Contract。
- 新实现必须从当前 Catalog 与安全边界重新设计和验收。

## Terms

- **Egress:** 自动化向外部网络发起请求的受控能力。
- **Node Contract:** Compiler、runtime、编辑器和文档共同消费的版本化节点事实。
