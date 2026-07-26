---
title: 定义 Node Package 与 Plugin Host 协议
label: wayfinder:prototype
parent: ../map.md
status: open
assignee:
blocked_by:
  - define-data-types-and-value-envelope.md
  - define-node-contract-metaschema.md
  - define-capability-and-target-planning.md
  - define-blob-store-streams-and-resource-broker.md
---

# 定义 Node Package 与 Plugin Host 协议

## Question

Node Package manifest、artifact identity、host API version、Process Node RPC、Wasm imports/exports、Value Envelope、状态/错误流、取消、超时、日志、capability broker 与生命周期应如何组成一个可在 Windows 首先落地、又不绑定 Go 的最小端到端协议？产出应包含一个 Process Node 和一个 Wasm Node 原型。
