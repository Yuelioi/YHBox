---
status: active
when_to_read: 改子图折叠 (useFolding) / virtual marker (SubgraphInput·Output) 接线 / 改 pinsFor / 撞「子图入口出口线掉节点底部」或 INVALID_PIN Subgraph 不存在 in pin
applies_to: [subgraph, fold, virtual-marker, pin-naming, vue-flow, useFolding, pinSpec, runtime-seed]
last_updated: 2026-06-04
recurrences: 1
---

# 子图 virtual marker 的 pin 名必须三层一致 (渲染 / 边 / runtime)

**Symptom**: 折叠选中节点为子图后, ① 主图保存失败 `节点 n-call_xxx (Subgraph) 不存在 in pin in, 还有 2 个错误`; 修了边 pin 名后 ② 子图入口/出口的线**不从 handle 出来、掉到节点底部**。

**Root cause** (I assumed pin 名随便取 / 改 `PIN_SPECS` 就能改渲染, 实际三层各有约定且渲染根本不读 PIN_SPECS):

subgraph 的 entry/output 是 **virtual marker** (`SubgraphInput`/`SubgraphOutput`, 不在 `graph.nodes`、不在 backend registry)。它们的 exec pin 名必须**三层一致**, 否则各层各崩:

1. **存的 edge pin** — `useFolding.ts` 原硬编码小写 `in`/`out`。实际全仓走 **PascalCase exec 约定**: Subgraph 调用节点 exec-in = `In` (`subgraph.go` sgInExec); 入口 marker exec-out = `Done` (runtime `dispatch_v5.go` 从 `entryID+".Done"` 播种); 出口 marker exec-in = `In`; **边界节点用各自真实 pin** (WindowTarget 是 `In`/`Done`, 不是 in/out)。错了 → validator 报 `INVALID_PIN`、主图存不下 (markers 本身是 virtual 不被 validator 查, 报错的是父调用节点 + 边界真实节点)。
2. **渲染 handle** — `ContainerFlowNode` 用 `pinsFor(kind)`, 它查 **registry (`getSpec`)**, marker 未注册 → 落 `{execIn:['in'],execOut:['out']}` **fallback**。`PIN_SPECS` 那张表手填了 marker 的 pin 但 `pinsFor` **根本不读它** → handle 画成 in/out, 跟 `Done`/`In` 的边对不上, vue-flow 把线甩到节点底部。
3. **runtime 播种** — `r.edges.next(entryID+".Done")` 逐字读边、无 pin 改写器 (rewriter 只改 nodeID 不改 pin)。所以存的边字面必须是 `.Done`。

**Lesson**: 碰子图折叠 / marker 接线前, 先 grep 三层确认 pin 名: runtime `entryID+".Done"` (dispatch) + `subgraph.go` 的 `sgInExec/sgOutDone` + `pinsFor` 的渲染路径。marker 不在 registry → **必须确认 `pinsFor` 真从 `PIN_SPECS` 取, 而非落 in/out fallback** (本次就是改了 PIN_SPECS 却没改 pinsFor, 改动全程 inert)。别信「改了那张表就生效」—— grep 谁真读它。边界节点端**保留原 external 边的真实 pin**, 别重建成 in/out。
