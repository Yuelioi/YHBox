---
kind: trap
summary: "Vue Flow nodes/store 双向同步是 shallow：拖拽坐标读 event.node.position，外部 nodes 重建还会用旧 position 覆盖内部实时位置。"
activation: symptom
read_when: "写 Vue Flow drag/position/selection handler，修改 nodes/v-model/store 同步，或遇到点击、连线、Source 刷新后节点位置回跳"
recheck_when: "升级 @vue-flow/core，或改变 WorkflowEditorView 的节点投影与 selection 所有权时"
---
# Vue Flow nodes 与内部 store 的浅同步陷阱

## 教训

Vue Flow 1.48.2 的 nodes 双向 watcher 只监听 nodes ref 和 length。拖动期间内部只 mutate state.nodes[i].position，不会把元素深层位置同步回外部数组。

因此：

- drag/drag-stop 的实时坐标只能读 event.node.position；其他事件可读 useVueFlow().findNode(id)?.position。
- 不能从外部 flowNodes 找拖拽后坐标，它仍是拖动前值。
- 外部 computed nodes 若因 selected、面板状态或 Source 命令重建，会触发 store.setNodes；parseNode 会 Object.assign 外部旧 position 到已有内部节点，直接覆盖手势实时位置。
- selection 不应通过重建整组 nodes 维持。让 Vue Flow 拥有瞬时选择态，领域 EditorSession 只拥有持久内容。
- 拖拽期间若 Source 可能刷新，节点投影必须用 keyed live-position overlay；drag-stop 用 event.node.position 提交一次领域命令，再清 overlay。

## 反模式

```ts
const flowNodes = computed(() =>
  sourceNodes.value.map((node) => ({
    id: node.id,
    position: node.position,
    selected: node.id === selectedNodeId.value,
  })),
)

function onDragStop(event: NodeDragEvent) {
  const stale = flowNodes.value.find((node) => node.id === event.node.id)
  save(stale.position)
}
```

这里 selection 变化会重建 nodes 并把旧 position 写回内部 store；drag-stop 又会保存旧值。

## 正确边界

- 瞬时手势/选择：Vue Flow store。
- 拖拽中的跨刷新坐标：live-position overlay，以 event.node.position 更新。
- 持久位置：Workflow Source / EditorSession，只在 drag-stop 提交一次。
- 点击与拖拽用显式 dragHandle 分隔；端口和交互控件保持 nodrag。

## Case 1：snap 读取了外部旧位置

旧编辑器 onSnapNodeDragStop 从 flowNodes 取 dragged 坐标，导致引导线正确但松手不吸附。根因是 store 深层 mutation 不会同步回外部 nodes；改为 event.node.position 后修复。

此前两次尝试错误地归因于提交竞态或 updateNode 权威性，直到读取 @vue-flow/core 的 watchNodesValue 才确认浅 watcher 行为。遇到框架同步问题，多次尝试无效后必须读已安装源码。

## Case 2：selection / Source 刷新覆盖实时位置

3.1 WorkflowEditorView 的 computed nodes 同时包含持久 position 与 selected。真实 Vue Flow store 回归测试先把内部节点移到 (320,240)，再只改变 selection；watchNodesValue 调用 setNodes 后位置稳定回跳到外部旧值 (40,60)，连续三次复现。

修复将 selection 从外部节点投影移除，并加入 live gesture position overlay。selection 变化不再重建节点；拖拽期间即使 Source 因其他命令刷新，外部投影仍携带实时位置。节点 dragHandle 同时收窄到 header，避免默认 1px threshold 把正文点击变成微拖拽。
