---
title: 定义 Program 3.1 与 Run 语义
label: wayfinder:grilling
parent: ../map.md
status: open
assignee:
blocked_by:
  - define-node-contract-metaschema.md
  - define-capability-and-target-planning.md
  - define-blob-store-streams-and-resource-broker.md
---

# 定义 Program 3.1 与 Run 语义

## Question

Compiler 应把 Data、Exec、Error、event、region、subgraph、disabled、retry、cancel、timeout、recorded value 与 lineage lower 成什么不可变 Program 事实，runtime 又应提供多小的 interpreter interface，才能彻底删除按 node kind 分支的 ContainerRunner 并保证调试与正常执行同义？
