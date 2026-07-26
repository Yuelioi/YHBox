---
title: 定义 Node SDK、文档生成与 Conformance 工具链
label: wayfinder:grilling
parent: ../map.md
status: open
assignee:
blocked_by:
  - define-node-contract-metaschema.md
  - define-node-package-and-plugin-protocol.md
  - prototype-schema-driven-authoring.md
---

# 定义 Node SDK、文档生成与 Conformance 工具链

## Question

应从 Node Contract、Data Type 与 Authoring Projection 生成哪些 Go/TypeScript/WIT/Protobuf bindings、schema、编辑器模型、节点参考文档、示例包和 conformance vectors，才能让内置节点、Wasm Node、Process Node、UI、AI/MCP 与文档站点共享同一事实来源？

还需定义 `new/lint/test/package/diff` 开发命令、breaking-contract 检测、生成物追溯信息、CI drift gate，以及一个第三方作者无需修改 Yotta 中央 switch 就能完成节点开发和验证的最小路径。
