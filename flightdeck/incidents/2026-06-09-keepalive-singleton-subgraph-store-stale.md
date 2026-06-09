---
status: active
when_to_read: 撞容器编辑器里 Subgraph 节点渲染 "(子图未找到)" 但子图磁盘上/后端 ListSubgraphs 明明有; 在多个容器编辑器间切换后某容器独有的子图节点失效; 改 useContainerEditorStore (subgraphsForCurrentContainer / activeContainerID / editorPath 这些全局单例) 或给 <keep-alive> 缓存的编辑器加状态前; 设计"当前容器"类全局单例时
applies_to: [frontend, vue, keep-alive, pinia, singleton-store, container-editor, subgraph, onActivated, useContainerDraft, ContainerFlowNode]
last_updated: 2026-06-09
resolved_by: 9ccccbf (useContainerDraft onActivated 重置子图 store)
---

# keep-alive 缓存编辑器共享全局单例子图 store 致跨容器"(子图未找到)"

## Signature
- symptom: `Subgraph 节点出口渲染成 "(子图未找到)" (node.Subgraph.fallback_missing); 子图在该容器磁盘 + 后端 ListSubgraphs 都有`
- error_type: —  (前端状态/单例污染, 非 exception)
- where: frontend useContainerEditorStore.subgraphsForCurrentContainer (全局单例) ← ContainerFlowNode.vue resolveSubgraphCallExecOut; useContainerDraft 只在 onMounted 写它
- trigger: `<keep-alive include="ContainerEditorView">` 同时缓存多个容器编辑器, 在它们之间切换 (切到别的容器再切回)

## 症状/复现

A 容器折叠/建一个**只属于 A** 的子图 (sg-X) → 切到 B 容器编辑器 → 切回 A → A 画布上引用 sg-X 的 Subgraph 节点变 "(子图未找到)"。分享/导出该子图也像"失败"(实测后端 export 成功, 是前端节点显示坏了, 用户感知"一分享就没了")。

**障眼法**: 若 sg 在 A、B 里**同 id** (B 从库导入过同一个), 两个列表都有 → 永远查得到 → 看着像"跨容器同步、很正常"; 只有**某容器独有**的子图才暴露。模板/ClickTemplate 与否无关。

## 根因

`useContainerEditorStore` 的 `subgraphsForCurrentContainer` / `activeContainerID` / `editorPath` 是**全局单例一份**, 隐含假设"同时只有一个当前容器编辑器"。但:
- `<keep-alive include="ContainerEditorView" :max="3">` 同时缓存最多 3 个容器编辑器实例 (见 App.vue)。
- 每个实例只在 `onMounted` (每实例一次) 调 `setActiveContainer` 写单例; **切回缓存实例时 onMounted 不再跑**, 没有 `onActivated` 重同步。

于是: 开 A (单例=A 子图) → 切 B (B 挂载, 单例被覆盖成 B 子图) → 切回 A (缓存命中, onMounted 不跑, 单例仍是 B 子图)。`ContainerFlowNode.vue` 直接读单例 `subgraphsForCurrentContainer.find(s => s.id === SubgraphID)`, A 独有的 sg-X 不在 B 列表里 → "(子图未找到)"。

同族: 跟 [[2026-06-09-import-bypasses-container-store-cache]] **同症状 "(子图未找到)" 但不同根因** —— 那条是后端 import 绕过容器 Store 内存缓存; 这条是前端全局单例被 keep-alive 兄弟实例覆盖。排查 "(子图未找到)" 先分清是哪一个。

## 修法

`useContainerDraft` 加 `onActivated`: keep-alive 重新激活时用**本容器**磁盘子图 `setActiveContainer` 重置单例; 残留的 `editorPath` 头节点不在本容器子图里就 `resetPath` 回主图 (否则 activeGraph 算空白)。首次挂载 (draft 未就绪) 跳过, 交给 onMounted。抽 `toSubgraphSummaries` 去重 load/refresh/activate 三处映射。

**未根治 (用户已知, 留待子图系统集中重构)**: 单例模型本身的债 —— `editorPath` 跨容器共享、子图未存盘编辑在切容器时被覆盖 (mergeSubgraphs 注释提过)、id 碰撞时 mergeSubgraphs 跨容器取错版本。彻底解法是把这些状态**按容器隔离** (map keyed by containerID) 而非全局单例 + onActivated 补丁。onActivated 只保证"可见的编辑器看到对的列表"。

## Cases
- 2026-06-09 首次 (用户报: 容器2 折叠 ClickTemplate 成子图后分享 → "(子图未找到)"; 实为切到容器3 再回容器2 后单例污染)
