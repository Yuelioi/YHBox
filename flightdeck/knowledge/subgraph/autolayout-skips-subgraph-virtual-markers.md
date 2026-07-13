---
kind: trap
summary: "子图入口/出口不在 graph.nodes；全图操作必须合入 marker，异步结果还必须绑定发起时的 editor context"
activation: symptom
read_when: "写任何遍历子图 graph.nodes 的全图操作(自动布局/导出/转脚本/遍历/分析)，或让异步工作读取 activeGraph/editorPath 后再写回时"
recheck_when: "activeGraph/editorPath 的容器隔离方式、ELK 加载/worker 边界或 applyDraftMutation 历史语义改变时"
---
# ⚠ 全图操作漏掉子图 virtual marker (入口/出口不在 graph.nodes)
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
(防用户误删/改 kind; 见 [subgraph-marker-pin-convention.md](subgraph-marker-pin-convention.md))。边按 `<nodeID>.<pin>` 引用它们。

任何"全图操作"若只遍历 `graph.nodes` 就会漏掉这俩。本例 `autoLayout` 读 `activeGraph.nodes`
(只 body) → 两层连锁: ① marker 不进 ELK, x/y 不重排; ② `buildElkGraph` 的 `elkEdges` 过滤
`layoutIds.has(两端)` 把「body↔marker」的边全丢了 → body 丢约束, 自己也排乱。

**通用教训**: 子图的"完整图"= `graph.nodes` + entry + outputPins。导出 / 转脚本 / 遍历 / 分析 /
布局 等任何全图操作都得把 marker 并进来, 别假设 `graph.nodes` 是全部。读 draft 子图内容的
姊妹坑见 [draft-subgraphs-phantom-field.md](draft-subgraphs-phantom-field.md)。

## 修法

`useElkLayout.autoLayout` 子图层级: 用 `subgraphMarkerNodes(sg.entry, sg.outputPins)` 合成 marker
pseudo `GraphNode`, 跟 body 一起喂 `buildElkGraph`(marker 被排版 + 连它的边不再被丢); 布局后按 id
路由写回 —— 真实节点写 `graph.nodes`, marker 用 `writeMarkerPositions` 写回 `sg.entry`/`outputPins`
+ `touchSubgraph` 标脏(跟 `onNodesChange` 拖动 marker 同写回路径)。两个纯函数在 `elkGraph.ts`, 有单测。

ELK engine 动态加载与 `elk.layout()` 都会跨 `await`。发起时必须同时捕获 graph identity、container ID、editor path、subgraph/marker owner，并在每个 await 后验证；上下文已切换就静默丢弃，不能重新读取当前 `activeGraph` 后把旧结果写进去。最终 graph 与 marker 写回必须位于同一次 `applyDraftMutation`，这样子图快照可一次 undo。

## Cases
- 2026-06-13 首次: 子图自动布局漏排入口/出口 marker + body 乱。修: marker 合进 ELK 布局 + 写回 metadata。
- 2026-07-13: ELK 改 lazy import 后暴露异步切图竞态。修: capture/validate layout context，并增加 engine-load 与 layout-await 中途切图测试。
