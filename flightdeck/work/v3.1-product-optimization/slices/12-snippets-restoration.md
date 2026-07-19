---
slice: "12"
title: Snippets 恢复
status: completed
---

## Outcome / Question

删除写死的“常用配方”，在 3.1 架构上恢复 3.0 Snippets 的用户心智模型：用户可把一个已经配置好的节点保存为可管理、可搜索、可重复插入画布的个人模板。

## Completion criterion

- 工作流目录不再展示“常用配方”，相关硬编码 recipe builder、入口、i18n 和测试被完整删除。
- 节点上下文菜单支持“保存为 Snippet”；保存精确 NodeRef、可编辑 config 与安全的 target slot 引用，不保存 Node ID、位置、运行 Grant、credential secret 或宿主 handle。
- Snippet 具有名称、描述、分类、标签以及创建/修改/最近使用时间；可搜索、筛选、编辑、删除。
- 编辑器左侧资源工作区提供独立 Snippets 入口；单击插入视口中心，拖拽插入落点，生成新的 Node ID 并保持原 Snippet 不变。
- Snippets 经 durable application service 持久化，不复制 3.0 的 localStorage；损坏单项被隔离并给出可操作错误。
- 保存、重开应用、插入、编辑、删除的真实用户旅程闭环；与代码编辑器 Code Snippet、资产和子图区分清楚。

## Blocked by

- Slice 11 先恢复稳定画布相机手势，避免 Snippet 拖放与插入验收被画布交互故障干扰。
- 3.0 行为取证只作为 UX 基线；实现必须使用 3.1 Workflow Source、Node Contract 和稳定 application service。

## Verification

- 后端 service 覆盖 schema validation、原子保存、损坏隔离和 CRUD。
- 前端覆盖 Snippet 元数据、精确节点快照、插入时新 ID/位置、搜索筛选与删除确认。
- WebView 中保存一个带配置节点为 Snippet，切换工作流后搜索并拖入画布，确认配置保留且可独立编辑。
- Stage G 末统一运行聚合测试、`task check`、真实 WebView smoke 和 production build。

## Out of scope

- 恢复 3.0 Container runtime、localStorage 持久化、旧 node kind 分发或代码编辑器 Code Snippet。
- 第一阶段把多个节点及边界连线保存为 Snippet；多节点复用继续使用 Source-native 子图或后续显式扩展。

## Result

已完成。硬编码“常用配方”入口、builder、i18n 与测试已删除；新增 `internal/services/snippet` durable 深模块和 Wails RPC，单项存放于 `data/snippets/`，通过原子写入、schema 校验、敏感运行字段拒绝和损坏单项隔离守住边界。

编辑器支持从节点右键保存 Snippet，管理名称、描述、分类和标签，在独立 Snippets 资源面板中搜索、筛选、编辑、删除、单击插入视口中心或拖放到指定落点。保存内容保留精确 NodeRef、config、bindings 和 disabled，但不保留节点 ID、位置、Grant、credential、secret 或宿主 handle；插入时校验当前 Catalog 契约、生成新 ID，并持久记录使用次数与最近使用时间。

真实 Wails/WebView journey 已覆盖保存、持久化展示、点击插入、视觉选中、删除和继续导航；产物位于 `.task/workflow-editor-smoke/20260719-235101/`。Stage G 聚合验收 `task check` 通过。
