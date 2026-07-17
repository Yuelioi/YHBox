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

无。

## Verification

阶段内已运行 pnpm vitest run src/app/editor/workflowFlowProjection.spec.ts（2/2）和 pnpm typecheck。Stage 1 末统一运行完整门禁和 Windows GUI 手势矩阵。

## Out of scope

候选菜单、多选、布局和调试。

## Result

Completed。真实 Vue Flow store 测试证实：外部 computed nodes 同时投影 selected 和旧 position 时，selection 变化会触发 setNodes，把内部实时位置从 (320,240) 回写为 (40,60)。修复移除外部 selected 投影，以 Vue Flow 为选择态事实源；加入 live gesture position overlay 抵御拖拽期间 Source 刷新；dragHandle 收窄为节点 header。真实红灯连续三次稳定失败，修复后 selection/source-refresh 两个用例转绿且 typecheck 通过。
