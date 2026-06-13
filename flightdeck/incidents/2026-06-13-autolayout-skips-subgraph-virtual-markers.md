---
status: active
when_to_read: 写任何遍历子图 graph.nodes 的全图操作(自动布局/导出/转脚本/遍历/分析)前; 撞「子图入口出口没被处理/位置不对/漏了这俩节点」
applies_to: [frontend, subgraph, virtual-marker, auto-layout, graph-traversal, entry, outputPins, internal/services/container/model.go, frontend/src/composables/containerEditor/useElkLayout.ts, frontend/src/composables/containerEditor/elkGraph.ts, frontend/src/composables/containerEditor/useContainerDraft.ts]
last_updated: 2026-06-13
resolved_by:
---

# 全图操作漏掉子图 virtual marker (入口/出口不在 graph.nodes)

## Signature
- symptom: 子图里运行自动布局, 入口/出口 marker 不跟着排版(留在原地)、body 布局还变乱
- error_type: —  (数据遗漏/布局错, 非异常)
- where: `useElkLayout.autoLayout` + `buildElkGraph` 只吃 `activeGraph.nodes`; marker 存 `model.go` `SubgraphMarker`/`SubgraphOutputDecl`, 不在 `Graph.Nodes`
- trigger: 全图操作只遍历 `graph.nodes`, 漏了 virtual entry/output marker

## 症状/复现

进一个子图 → 自动布局 → 入口(SubgraphInput)/出口(SubgraphOutput)不动、停在原位; body 节点
重排后跟入口/出口错位, 整体看着乱。

## 根因

子图入口/出口是 **virtual marker** —— 存在 `subgraph.entry`(`SubgraphMarker`) 和
`subgraph.outputPins[]`(`SubgraphOutputDecl`), 各自带 `nodeID/x/y`, **刻意不进 `graph.nodes`**
(防用户误删/改 kind; 见 [[2026-06-04-subgraph-marker-pin-convention]])。边按 `<nodeID>.<pin>` 引用它们。

任何"全图操作"若只遍历 `graph.nodes` 就会漏掉这俩。本例 `autoLayout` 读 `activeGraph.nodes`
(只 body) → 两层连锁: ① marker 不进 ELK, x/y 不重排; ② `buildElkGraph` 的 `elkEdges` 过滤
`layoutIds.has(两端)` 把「body↔marker」的边全丢了 → body 丢约束, 自己也排乱。

**通用教训**: 子图的"完整图"= `graph.nodes` + entry + outputPins。导出 / 转脚本 / 遍历 / 分析 /
布局 等任何全图操作都得把 marker 并进来, 别假设 `graph.nodes` 是全部。读 draft 子图内容的
姊妹坑见 [[2026-06-12-draft-subgraphs-phantom-field]]。

## 修法

`useElkLayout.autoLayout` 子图层级: 用 `subgraphMarkerNodes(sg.entry, sg.outputPins)` 合成 marker
pseudo `GraphNode`, 跟 body 一起喂 `buildElkGraph`(marker 被排版 + 连它的边不再被丢); 布局后按 id
路由写回 —— 真实节点写 `graph.nodes`, marker 用 `writeMarkerPositions` 写回 `sg.entry`/`outputPins`
+ `touchSubgraph` 标脏(跟 `onNodesChange` 拖动 marker 同写回路径)。两个纯函数在 `elkGraph.ts`, 有单测。

已知限制: marker 坐标不进 undo 快照(子图池不在 draft 里, `snapshotDraft` 看不见) —— 跟"拖动 marker
也不进 undo"一致, 非本次回归。

## Cases
- 2026-06-13 首次: 子图自动布局漏排入口/出口 marker + body 乱。修: marker 合进 ELK 布局 + 写回 metadata。
