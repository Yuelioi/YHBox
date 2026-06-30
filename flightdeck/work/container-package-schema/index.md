# 容器包 schema 重设计

SUMMARY: 破坏式重设计容器存储模型, 让本地容器从创建第一天就是可投稿/可发布/可更新的包。

READ WHEN: 改容器持久化、容器列表字段、在线容器、投稿/导入/更新、依赖闭包、target/AI 本机绑定前。

## 状态

- 已完成底层节点调研。
- 已定设计方向: `package.json` + `graph.json` + `installation.json` + `yotta-lock.json`。
- 已完成阶段 1: 后端模型类型和 `Graph.schemaVersion` JSON 形状。
- 已完成阶段 2 基础: 依赖闭包拆为 templates/clips/subgraphs, 并新增 `yotta-lock.json` 构建函数。
- 下一步阶段 3: Store 改为 `package.json` + `graph.json` + `installation.json` + `yotta-lock.json` 多文件目录。

## 文件

- [design.md](design.md) — 容器包 schema 设计。
- [plan.md](plan.md) — 破坏式落地执行计划。

## 相关知识

- ../../knowledge/nodes/node-system-architecture.md
- ../../knowledge/nodes/script-system.md
- ../../knowledge/subgraph/asset-subsystem.md
- ../../knowledge/build/build.md
