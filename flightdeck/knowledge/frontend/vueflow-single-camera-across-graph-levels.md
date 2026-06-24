# ⚠ vue-flow 全程单相机, 切逻辑图层级要自己存取 viewport

SUMMARY: vue-flow 全程单相机, 切逻辑图层级要自己存取 viewport.
READ WHEN: 改容器编辑器视口/相机/切主图↔子图层级行为前; 撞「切图后相机跑飞 / 视口不对 / 首次进入看不到内容」

---


## Signature
- symptom: 在内容很远的子图 pan 走(如 19000,1000), 切回主图相机还停在那 → 主图内容(3000 区域)看不见, 像"跑飞"
- error_type: —  (UI/视口, 非异常)
- where: `ContainerEditorView` 的 vue-flow 相机; `<VueFlow fit-view-on-init>` 只首次挂载 fit 一次
- trigger: 切逻辑图层级 (push/pop `editorPath`) 不存/取 viewport

## 症状/复现

主图 → 进子图 → 在子图里 pan/zoom 到远处 → 切回主图: 相机停在子图那个远坐标, 主图内容不在视野。

## 根因

vue-flow 全程**只有一个相机**。`fit-view-on-init` 只在首次挂载 fit; 之后切图层级只 `syncFlowFromDraft`
重渲染节点, **完全不动相机**。所以主图/各子图共享同一个 viewport, 切层级时相机原地不动 → 看着乱/飞。
逻辑上的"多张图"在 vue-flow 眼里是同一块画布换了 nodes。

## 修法

per-(容器, 图层级) 缓存 viewport (`viewportByContainer`, 视图态按容器隔离, 跟 `editorPathByContainer` 同规矩,
`dropContainer` 一起清)。切层级时: 同步存旧层级 `getViewport()`, 取目标层级缓存 → `setViewport` 恢复;
**首次进入(无缓存)→ 聚焦起始节点**(子图=入口 marker `sg.entry` / 主图=唯一 `Start` 节点, `centerOnNode` zoom 1),
找不到才 `fitView` 兜底 (内容很大的图 fitView 会缩成一小团, 落在入口更可用)。层级 key = sgID 或 `MAIN_GRAPH_KEY`。

同属"子图是 virtual / 全图操作要单独处理"族, 见 [[2026-06-13-autolayout-skips-subgraph-virtual-markers]]。

## Cases
- 2026-06-13 首次: 切子图后切回主图相机跑飞。修: per-层级 viewport 缓存 + 首次进入聚焦入口。
