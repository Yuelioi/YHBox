# Slice 21 — 权威类型系统基础

## Outcome / Question

Catalog 驱动的 Type System 是否已成为 Compiler 和 Authoring 判断具体类型关系与 constraint 的唯一权威，并保证 runtime 不做隐藏 coercion？

## Completion criterion

名义类型关系、trait、LUB、泛型实例求解与 Projection parity 全部由 sealed Catalog 驱动；Integer、领域类型和 List 的边界有回归覆盖。

## Blocked by

无。

## Verification

Integer/Number、领域隔离、list invariance、constraint、LUB、关系歧义与绑定顺序测试，以及 2026-07-18 完整 `task check`。

## Out of scope

类型能力闭包、结构化 Break 节点和最终交互体验，由 Slice 22–23 承担。

## Result

Completed。

- Data Type semantic：closed traits、显式 assignable targets、digest 固定、引用/环/未知 trait 校验。
- Type System：exact/assignable、trait membership、union、list invariance、唯一 least-upper-bound。
- Compiler solver：节点实例 scope、constraint 执行、occurs check、重复证据合并和顺序无关绑定。
- Integer→Number 显式安全提升；duration/key-code 等领域类型不按 schema primitive 自动兼容。
- Projection 输出 traits 与 assignable closure；Go/TS/generated Catalog parity 通过。
- EditorSession 图级固定点实例专化覆盖 StateRead<T>、Select<T>、ToString<T> 等链路。
