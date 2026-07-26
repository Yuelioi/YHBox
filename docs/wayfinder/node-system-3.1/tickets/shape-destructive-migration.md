---
title: 规划节点系统 3.1 的破坏性迁移
label: wayfinder:grilling
parent: ../map.md
status: open
assignee:
blocked_by:
  - define-program-and-run-semantics.md
  - define-node-package-and-plugin-protocol.md
  - define-node-package-lifecycle-and-trust.md
  - define-node-sdk-doc-generation-and-conformance.md
  - prototype-schema-driven-authoring.md
---

# 规划节点系统 3.1 的破坏性迁移

## Question

在不保留兼容层的前提下，如何把 Source、137 个内置节点、Compiler、runtime、frontend、MCP、脚本、fixtures、文档和跨平台 adapter 切成可持续通过门禁的 tracer-bullet 迁移顺序，并明确旧模块的删除条件与最终验收矩阵？
