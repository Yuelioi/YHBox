# Slice 1：交互正确性与位置同步

## Outcome / Question

单击、选中、拖线、取消拖线和操作节点控件都不会移动节点；真实拖拽只提交一次最终位置。先证实跑偏链路再修复。

## Completion criterion

- 追踪 gesture id、Vue Flow 内部位置、Workflow Source 位置、选择变化和 move-node。
- 覆盖 0/1/2+ 像素、缩放/DPI、已选/未选、各 handle、有效/无效/空白落点。
- 点击/连线产生 0 条 move-node；真实拖拽产生 1 条最终 move-node。
- 只有 header 可拖；handles、按钮、输入和滚动区 nodrag。
- 重载位置一致，undo/redo 各恢复一次。

## Blocked by

无；最高概率是假设中的 computed nodes 与内部 store 浅同步竞态，必须用证据确认。

## Verification

只跑交互层定向测试与必要 typecheck；Stage 1 末统一完整验收。

## Out of scope

候选菜单、多选、布局和调试。

## Result

Planned。目标结构：稳定 Vue Flow 交互投影 + EditorSession 持久位置事实源。
