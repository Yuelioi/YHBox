# ⚠ 成功反馈一律内联, toast 只留错误/后台事件

SUMMARY: 成功反馈一律内联在触发点, toast 只留错误/后台事件.
READ WHEN: 加复制/保存/导入等成功反馈 / 想 toast.add 一条成功消息 / 写改 .vue 的用户反馈触发点时

---


## Signature
- symptom: 给复制/保存等成功操作加了 `toast.add({ color: 'success' })` 顶部弹出成功提示
- error_type: —  (UI 约定违反, 非异常)
- where: frontend `.vue` 触发点 (本次 NodeInspector.vue 复制按钮); 规则正源 `checklists/ui.md`「反馈方式」
- trigger: 写/改 `.vue` 前没读 ui.md, 凭直觉给成功操作配了 toast

## 症状/复现

加一个复制按钮, 顺手写 `toast.add({ title: '已复制', color: 'success' })`。用户明确不喜欢成功类
toast —— 顶部弹出干扰视线、和操作点割裂。成功反馈本该内联在触发点。

## 根因

改 `.vue` 前**没读** [checklists/ui.md](../checklists/ui.md)(它的 `when_to_read` 就是「写/改 .vue 前」)。
ui.md「反馈方式」决策树早有定论, 漏读 → 凭直觉造方案。本质是头号铁律「有源码/有约定必读」在前端
约定上的同款踩法 —— 跟 [[2026-05-26-snippet-drawer-debug-discipline]] 一类纪律问题。

## 修法

按 ui.md「反馈方式」决策树:

1. **结果当前视野立即可见**(列表项消失 / 画布重排 / 详情图更新) → **什么都不弹**, 变化本身就是反馈。
2. **复制到剪贴板**(不可见): **优先原地, 别 toast**。持久按钮 → 内联闪 ✓(`copied` ref ~1500ms,
   icon 切 `i-tabler-check`, 文案 `common.copied`); **下拉菜单项** → `onSelect(e)` 里 `e.preventDefault()`
   留住菜单 + 被点项原地切「已复制 ✓」(别因为"菜单会关"就退 toast); 真留不住才用短 toast(`duration: 1500`)。
3. **保存类持久按钮** → 内联「已保存 ✓」(参 ContainerEditorToolbar `saveFlash`)。
4. **后台/跨窗/系统替你做的事**(运行入队 / 导出分享 / 录制落盘 / 自动重连 / 校准变更) → 保留 toast(正当用途)。
5. **错误/警告** → 一律 toast。

范例: `NodeInspector.vue` 复制下拉(ID/JSON/脚本信息) — `onSelect(e)` 里 `e.preventDefault()` 留住菜单 +
被点项原地切「已复制 ✓」; `TemplateDetailPanel` / `SubgraphPropsPanel` 的 `onCopyID` — 持久按钮内联闪 ✓;
`ContainerEditorToolbar` 保存 `saveFlash`(§3)。**核心: 能原地就原地, toast 是留不住时的退路, 不是默认。**

## Cases
- 2026-06-13 首次: 给 NodeInspector 节点复制**按钮**(ID/JSON)加了成功 toast → 用户指出违反 ui.md 反馈树。
  先改持久按钮内联闪 ✓; 后因 icon 太小看不清, 收成一个复制 icon + 下拉菜单(ID/JSON/脚本信息);
  此时**误以为**「菜单项点完即关」只能 toast, 又弹了短 toast → 用户再次指出。最终: `e.preventDefault()`
  留住菜单 + 菜单项原地切「已复制 ✓」, 不 toast。**教训: 菜单能留住做原地反馈, 别预设它必关而退 toast。**
