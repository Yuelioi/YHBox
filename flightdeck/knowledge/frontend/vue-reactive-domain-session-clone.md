---
kind: trap
summary: "拥有 immutable JSON snapshot 且内部调用 structuredClone 的 class session 不能用 Vue reactive 深代理；用 shallowReactive 只追踪顶层事实替换。"
activation: symptom
read_when: "Vue/Wails 编辑器点击后报 DataCloneError；structuredClone 收到 Proxy；给 class/session/store 加 reactive；画布命令执行但节点不出现"
recheck_when: "EditorSession 改为原地深层 mutation；createEditorSession 响应式策略改变；structuredClone 边界改变"
---
# Vue 深代理不能跨 structuredClone 边界

`reactive(new EditorSession(...))` 会在读取 `session.source`、Catalog projection 和内部集合时返回 Vue Proxy。`EditorSession.apply()` 先对 Source 调 `structuredClone`，浏览器不能克隆 Proxy，因此目录点击在任何 Source mutation 前抛 `DataCloneError`；Vue Flow 没有机会收到新节点。

本仓 EditorSession 的事实更新采用顶层替换（例如 `session.source = next`），所以正确 seam 是 `createEditorSession()` 中的 `shallowReactive(new EditorSession(...))`：phase/source/diagnostics 等顶层赋值仍触发 computed，Source/Catalog/command 保持普通可克隆对象。不要在领域 session 中引入 Vue `toRaw`，那会把 UI 框架耦合进 authoring core，也只解决当前 clone 层级。

回归必须消费与生产视图相同的 `createEditorSession` factory，并同时断言 command 后 Source 节点数与 computed 投影节点数 `+1`。只测 raw `new EditorSession` 会漏掉这个集成错误。
