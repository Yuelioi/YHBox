# ⚠ vue-flow v-model:nodes ↔ store 同步是 shallow

SUMMARY: vue-flow v-model:nodes ↔ store 同步是 shallow, drag 后读坐标只能用 event.node.position.
READ WHEN: 写 vue-flow drag-stop / position-related handler / 改 store.nodes / v-model 同步逻辑 / drag 之后读 flowNodes 拿坐标但发现是老值

---


## 教训

vue-flow 1.48.2 用 `v-model:nodes="flowNodes"` 时, **store→model 同步 watcher 是 shallow**. 监听的是 `store.nodes` 这个 ref 跟 `nodes.value.length`. 拖动期间内部只 mutate `state.nodes[i].position` (element 内字段), 既不重赋值 ref 也不改 length → watcher 不触发 → `flowNodes.value[i].position` 拿不到新坐标.

**所以**: 在 `@node-drag-stop` handler 里要拿 dragged 节点的 live 坐标, **只能用 `event.node.position`** (这是 vue-flow store 的 GraphNode), **不能** `flowNodes.value.find(...).position` — 后者还是拖动前坐标.

Why: 读 [vue-flow-core.mjs:5807-5839](../../frontend/node_modules/.pnpm/@vue-flow+core@1.48.2_vue@3.5.34_typescript@6.0.3_/node_modules/@vue-flow/core/dist/vue-flow-core.mjs) 的 `watchNodesValue`. 双向 `watchPausable` 用 `[store.nodes, () => store.nodes.value.length]` 当 source — Vue 3 默认 shallow, 内部 element mutation 不触发. 反方向 `[models.nodes, ...]` 同样 shallow.

How to apply: 写 vue-flow drag / connect / 任何"拖动后想读节点坐标"的 handler:
- ✅ `event.node.position` (drag handler) / `useVueFlow().findNode(id)?.position` (其他 handler)
- ❌ `flowNodes.value.find(n => n.id === id)?.position`

通过 `@nodes-change="onNodesChange"` 走 `NodeChange[]` 携带的 position 也对 (vue-flow 直接 emit 每个变化, 不经 v-model).

## 反模式

```ts
function onSnapNodeDragStop(event: NodeDragEvent) {
  // ❌ 错: flowNodes 在 drag 中没被同步, .position 是拖动前
  const flowNode = flowNodes.value.find((fn) => fn.id === event.node.id)
  const draggedX = flowNode.position.x  // stale!

  // ✅ 对: event.node 是 vue-flow store 的 live 节点
  const draggedX = event.node.position.x
}
```

## Case 1 — 2026-05-28 snap 第三次尝试才真修

Editor 拖节点 snap 功能, [commit c9892c7](https://localhost/c9892c7) 跟 [b63bf12](https://localhost/b63bf12) 两次 fix 都没解决"引导线对但松手不 snap". 第三次 source dig 才看到根因:

`onSnapNodeDragStop` 用 `flowNodes.value.find(...).position` 拿 dragged 坐标, 算 `bestX/bestY` 跟其他节点比对. 但 `flowNodes` 在 drag 中没被同步 → `draggedX/Y` 是拖动**前**坐标. 用户拖到某节点旁边 (live 位置在 SNAP_EPSILON 内), 但 pre-drag 位置一般在 epsilon 外 → bestX/Y 都 null → snap 跳过 → 节点视觉停在 mouse-release 处.

对照 `onSnapNodeDrag` (drag 中, 引导线计算) 用 `event.node.position` — 这是为啥引导线显示对的.

**前两次 commit message 都把 root cause 写错了**:
- c9892c7: "vue-flow was racing the drag commit; routing through draft makes snap authoritative" — 误诊, 没接触双向 watcher pause/resume dance.
- b63bf12: "vue-flow has its own internal node store that does NOT re-read from the v-model array on every swap" — 方向反了 (实际是 store mutation 不同步到 v-model). `updateNode` hack 治标不治本, 因为 bestX/Y 计算用错源就根本算不出 snap target.

**第三次** vue-flow-store-vmodel-fix (dragStop 改用 `event.node.position`, 4 行 inline 改动, 未单独立 plan): dragStop 改用 `event.node.position` (跟 drag handler 同源). 1 处 4 行改动. typecheck 绿.

教训: 撞 framework gotcha 时, 改 2 次没 work → 第 3 次必须读 framework 源码, 不能继续在 commit message 写"应该是 X". CLAUDE.md 头号铁律的应用版.
