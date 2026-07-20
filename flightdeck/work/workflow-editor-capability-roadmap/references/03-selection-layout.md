# Slice 3：多选、对齐、分布与自动布局

## Outcome / Question

用户能快速整理中大型工作流，批量动作可预测、可撤销且不改变业务语义。

## Completion criterion

- 多选、框选、批量移动/Delete、复制/剪切/粘贴/复制节点。
- 两个以上节点支持六种对齐，三个以上支持水平/垂直等距分布；使用实际节点宽高。
- LR/TB 自动布局基于 ELK，保持选择、视口和布局前中心，异步结果带 graph/revision token。
- 吸附辅助线不会与布局或手势位置同步冲突，Alt 可临时反转。
- 批量动作通过统一上下文工具条和快捷键暴露。
- 每次批量动作只有一个 history 条目；未选范围和边语义不变。

## Blocked by

Slice 1。已解除。

## Verification

定向验证通过：EditorSession 原子 duplicate/move/delete、几何对齐/分布/吸附、真实 ELK 方向/中心锚定、Vue Flow 投影与 UI 接线共 5 个测试文件 24 项；pnpm typecheck、定向 oxlint、pnpm i18n:check 通过。

Stage 1 批量验收通过：

- 根目录 task check 退出码 0；全局 Go coverage 门槛、vet/staticcheck、前端格式/lint/typecheck/i18n/bindings、32 文件 123 项 Vitest、production bundle budget 全部通过。
- task build 生成完整 Windows 产物并通过。
- 正式 WebView smoke 实际完成 Start 根节点、目录点击/拖放、自动布局消除全部节点矩形重叠、Shift 框选 2 节点并显示批量工具条、点击空白清除选择、RunStarted 出口拖到空白处打开兼容候选菜单。
- 同一 smoke 验证暗色 controls/minimap、端口标签无重叠、Nuxt UI 退出确认、保存不弹成功 toast 而原位反馈、AI review 与资源库；人工查看 workflow-editor 与 assets PNG 均可用。
- smoke 自身用内存 CDP/WebSocket 状态机测试覆盖完整旅程，cmd/workflow-editor-smoke 覆盖率由 6.7% 提升到 63.6%。

## Out of scope

subgraph、reroute、注释框和跨 graph 布局。

## Result

Completed。Vue Flow nodes-change 维护多选事实，批量拖拽/Delete、复制/剪切/粘贴/复制节点均进入单个复合 EditorSession 命令并可一次 undo/redo；内部边与 node config/bindings/label/disabled 随 selection 安全复制。上下文工具条提供六向对齐、水平/垂直等距、复制/剪切/复制节点/删除和 LR/TB 自动布局；无选择时画布仍有一键布局入口。拖拽按实际尺寸对齐边/中心并显示吸附线，Alt 临时禁用。ELK 按需加载、保持旧图中心，并以发起时 Source 对象和 graph id 丢弃过期结果。

阶段验收发现 Vue Flow store 尺寸在布局触发点可能仍为空，旧 90px 兜底会让实际约 116px 的动态节点重叠；现优先读取节点 DOM 的未缩放 offsetWidth/offsetHeight，再回退 store 尺寸和常量，真实 WebView 重叠计数为 0。
