# 容器包 schema 重设计

SUMMARY: 破坏式重设计容器存储模型, 让本地容器从创建第一天就是可投稿/可发布/可更新的包。

READ WHEN: 改容器持久化、容器列表字段、在线容器、投稿/导入/更新、依赖闭包、target/AI 本机绑定前。

## 状态

- 已完成底层节点调研。
- 已定设计方向: `package.json` + `graph.json` + `installation.json` + `yotta-lock.json`。
- 已完成阶段 1: 后端模型类型和 `Graph.schemaVersion` JSON 形状。
- 已完成阶段 2 基础: 依赖闭包拆为 templates/clips/subgraphs, 并新增 `yotta-lock.json` 构建函数。
- 已完成阶段 3: Store 已改为 `package.json` + `graph.json` + `installation.json` + `yotta-lock.json` 多文件目录。
- 已完成阶段 4/5: Service/Store 聚合 DTO 暴露 package 字段, target/AI 本机绑定从 portable graph 拆到 installation, 前端列表支持分类筛选并对齐 `Graph.schemaVersion`。
- 已完成阶段 6 部分: MCP `save_container` 已返回 package 目录并写入四件套。
- 已完成阶段 6: MCP `save_container` 写新布局; Store/Service 可导出 `.yotta-container.zip`, 包含 `package.json`, `graph.json`, `yotta-lock.json`, 子图闭包、template 资产闭包和 clip 闭包, 排除 `installation.json`; 导出前强制校验 `yotta-lock.json` 未过期。
- 已完成前端导出入口: 容器卡片/列表行可选择保存路径并导出 `.yotta-container.zip`。
- 下一步: 阶段 7 全量收尾和人工检查。

## 文件

- [design.md](design.md) — 容器包 schema 设计。
- [plan.md](plan.md) — 破坏式落地执行计划。

## 相关知识

- ../../knowledge/nodes/node-system-architecture.md
- ../../knowledge/nodes/script-system.md
- ../../knowledge/subgraph/asset-subsystem.md
- ../../knowledge/build/build.md
