---
title: 定义 Blob Store、Stream 与 Resource Broker
label: wayfinder:grilling
parent: ../map.md
status: open
assignee:
blocked_by:
  - define-data-types-and-value-envelope.md
  - define-node-contract-metaschema.md
---

# 定义 Blob Store、Stream 与 Resource Broker

## Question

内容寻址 Blob Store 的 identity、storage、quota、lifecycle、cleanup、range read 与完整性校验，Stream 的背压、取消、终态和 producer/consumer cleanup，以及 Resource Broker token 的 authority、ownership、borrow/drop、expiry 与 crash cleanup 应如何统一定义，才能让 runtime、Wasm Node 和 Process Node 使用相同语义，同时不把路径、指针、平台句柄或进程内对象泄漏到 Value Envelope？

还需明确 blob、stream 与 handle 之间哪些转换必须成为画布可见、带 effect/capability 声明的节点，并规定其失败、配额和重放行为。
