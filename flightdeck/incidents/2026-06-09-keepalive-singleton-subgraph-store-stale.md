---
status: active
when_to_read: 撞容器编辑器里 Subgraph 节点渲染 "(子图未找到)" 但子图磁盘上/后端 ListSubgraphs 明明有; 在多个容器编辑器间切换后某容器独有的子图节点失效 / 拿错容器的 WindowTarget("没有异环窗口") / 模板截图定位错容器 / 莫名其妙的孤儿边; 改 useContainerEditorStore (subgraphsForCurrentContainer / activeContainerID / editorPath) 或 tplStore.containerId 等任何"当前/前台容器"全局指针前; 给 <keep-alive> 缓存的编辑器加状态前; 加任何新的"前台容器"全局指针时(必须 onMounted+onActivated 都设)
applies_to: [frontend, vue, keep-alive, pinia, singleton-store, container-editor, subgraph, onActivated, onMounted, foreground-pointer, useContainerDraft, ContainerFlowNode, templates-store, tplStore, capture, WaitTemplate, screen-picker]
last_updated: 2026-06-12
resolved_by: 20e25a9 (store 按容器隔离) + 66538fe (firewall: __missing__ 哨兵不进存盘边 + onActivated 重拉; 补 20e25a9 遗留) — 见下 Cases 复发#4
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

**根治 (20e25a9)**: 把 store 状态**按容器隔离** —— `subgraphsByContainer` / `editorPathByContainer` (keyed by containerID) 取代全局单一 ref。`activeContainerID` 降级成"哪个编辑器在前台"的指针, 各实例 `onMounted`(setActiveContainer) + `onActivated`(markActive 廉价翻指针) 指向自己; `onBeforeUnmount` dropContainer 释放 slot。对外 API (subgraphsForCurrentContainer / editorPath / pushPath...) 名字签名不变, 作用在 active slot 上, descendant 组件 + 单测零改动。`editorPath` 改只读 computed + `setPath` action (Pinia setup store 把 computed 当 getter, 原 `editorStore.editorPath=[...]` 直接赋值不可靠)。`useContainerDraft` 读改为按自己 containerID (myPath/mySubgraphs), activeGraph / dirty watch 不再被别的容器切前台误触发。

效果: 切到别的容器不再覆盖本容器 slot → 切回来子图/层级原样还在, 不再 "(子图未找到)"; 顺带消除 **未落盘子图编辑/导航层级切换丢失**、**id 碰撞 mergeSubgraphs 跨容器取错版本** 两个同源债。

历史: 先有补丁 9ccccbf (`onActivated` 切回时重新拉盘 + resetPath) —— 能止血但每次激活重 fetch、且切走时本容器未存盘编辑已丢。20e25a9 用按容器隔离根治后**替换**了它 (无需重 fetch / 无数据丢失)。教训: "当前 X" 类全局单例一旦遇上 keep-alive 多实例并存就破, 正解是按实例 key 隔离, 别用"激活时重置全局"打补丁。

## Cases
- 2026-06-09 首次 (用户报: 容器2 折叠 ClickTemplate 成子图后分享 → "(子图未找到)"; 实为切到容器3 再回容器2 后单例污染)
- 2026-06-10 **复发#4** (用户: 容器2 建子图+分享 → 容器3 用它 → **主图保存失败 "不存在 out pin `__missing__`"**)。
  两个叠加问题: (a) 20e25a9 按容器隔离后, `onActivated` 改成只 markActive **不重拉子图** —— 引出新 gap: 本容器
  keep-alive 后台期间被 import/分享写入新子图, 切回不重拉 → `subgraphsByContainer` 滞后 → 节点 `__missing__`。
  (b) **更深的真根, 前 3 次都没碰**: Subgraph 节点子图未解析的渲染兜底 `__missing__` 是**纯显示哨兵**, 却被
  `onConnect` 当真 pin 连成边、存进图 → 后端校验拒。**补根 66538fe (两层)**: ① **防火墙** —— `onConnect` 拦哨兵
  pin (`isSentinelPin`), 哨兵**永不落盘** → 根除"主图保存失败 out pin __missing__"这一类, **与列表是否滞后无关**
  (有 useGraphMutations.test.ts 单测); ② `onActivated` 切回时 `refreshSubgraphStore` (merge 式, 保留未落盘编辑、
  只补盘上新子图、只动本容器 slot) → 治滞后, 节点正常解析。真机验过。
  **教训**: 渲染兜底哨兵 (`__missing__`/`__empty__`) 必须**与持久化隔离** —— 显示可兜底, 但绝不能进存盘数据;
  否则任何瞬时 store 滞后都会被固化成坏图。前 3 次只堵"怎么别让子图丢", 没堵"哨兵漏进存盘", 所以反复。
- 2026-06-12 **复发#5** (用户: A 容器编辑中切到 B 容器, 编辑 B 时报"没有异环窗口"——拿的是**A 容器的 WindowTarget**;
  新建 WaitTemplate 截图后子图保存失败, 边引用不存在的节点 `waittemplate_*` 孤儿边)。
  **漏网的前台指针**: `tplStore.containerId` (templates.ts) 是**第二个**"前台容器"全局指针, `capture()` / `openScreenPicker`
  (GeometryWidget.vue:337) 靠它定位本容器目标窗口。它只在 `ContainerEditorView` 的 **onMounted** 设 (setContainer),
  **漏了 onActivated** —— keep-alive 切回已缓存容器时 onMounted 不跑, 指针停在上一个容器 → WaitTemplate 截图/校验拿错
  容器的 WindowTarget("没有异环窗口"), 节点没建成留下孤儿边 → 主图保存被 validator 拒。
  **修法 (本次)**: `onActivated` 也 `tplStore.setContainer(containerID)` —— 跟 `activeContainerID` 自己的
  onMounted+onActivated(markActive) pattern **同构**。这**不违背**复发#4 line 39 "别用激活重置打补丁"教训: 那条针对
  **per-container 数据**(re-fetch 丢未存盘编辑, 故按容器隔离); `tplStore.containerId` 背后**无 per-container 数据**
  (模板资产全局), 纯"前台容器指针", 激活重指即完整正解。
  **顺带防护**: dirty 守卫原只接 standalone 关窗; 嵌入态切容器(同路由改 param, `onBeforeRouteLeave` 不触发)/返回列表
  全无 dirty 拦截 → 未保存改动留在 keep-alive 缓存实例里助长串味。本次把守卫收进路由钩子 `onBeforeRouteLeave` +
  `onBeforeRouteUpdate`, 三态(保存/丢弃/取消), 丢弃 `reload()` 真正回盘。
  **教训(收紧)**: 任何"当前/前台容器"全局指针, **onMounted 设了就必须 onActivated 也设**(keep-alive 多实例下 onMounted
  只跑一次)。加这类指针前先分: 纯前台指针(→ onActivated 重指即可) vs 背后有 per-container 数据(→ 必须按容器 key 隔离)。
  第 5 次同根因复发(指针/数据各踩), **promotion 候选: 该升进 checklist 硬规则**。
