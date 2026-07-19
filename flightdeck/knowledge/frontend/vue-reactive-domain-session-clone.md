---
kind: trap
summary: "Vue reactive Proxy 不能直接跨 structuredClone 边界；class session 用 shallowReactive，外来 DTO 编辑状态用 shallowRef 并在 UI clone seam 解开外层 Proxy。"
activation: symptom
read_when: "Vue/Wails 编辑器或表单打开后黑屏；structuredClone 收到 Proxy；给 class/session/store/DTO 加 reactive；画布命令执行但节点不出现"
recheck_when: "EditorSession 改为原地深层 mutation；createEditorSession 或 DTO 编辑状态的响应式策略改变；structuredClone 边界改变"
---
# Vue 深代理不能跨 structuredClone 边界

`reactive(new EditorSession(...))` 会在读取 `session.source`、Catalog projection 和内部集合时返回 Vue Proxy。`EditorSession.apply()` 先对 Source 调 `structuredClone`，浏览器不能克隆 Proxy，因此目录点击在任何 Source mutation 前抛 `DataCloneError`；Vue Flow 没有机会收到新节点。普通 Wails DTO 也有同一边界：把 Schedule 放进深 `ref` 后再在子表单 `structuredClone(props.schedule)`，会在组件 setup 阶段直接抛错，表现为点击“新建计划”后内容黑屏。

本仓 EditorSession 的事实更新采用顶层替换（例如 `session.source = next`），所以正确 seam 是 `createEditorSession()` 中的 `shallowReactive(new EditorSession(...))`：phase/source/diagnostics 等顶层赋值仍触发 computed，Source/Catalog/command 保持普通可克隆对象。不要在领域 session 中引入 Vue `toRaw`，那会把 UI 框架耦合进 authoring core，也只解决当前 clone 层级。

对只在 Vue 表单中编辑的外来 DTO，使用 `shallowRef` 保存会话对象，避免父层自动深代理；如果值可能来自 Pinia reactive 列表，则只在 UI 的 clone seam 对外层值 `toRaw` 后再 `structuredClone`。这不是领域层解法，而是把框架适配留在组件边界。若 DTO 内部可能主动保存嵌套 Proxy，单层 `toRaw` 不足，必须重新审视 owner，而不是递归剥壳掩盖数据所有权问题。

EditorSession 回归必须消费与生产视图相同的 `createEditorSession` factory，并同时断言 command 后 Source 节点数与 computed 投影节点数 `+1`。DTO 表单回归必须直接传入 `reactive(...)` 值并挂载真实组件。只测 raw class 或普通对象都会漏掉集成错误。
