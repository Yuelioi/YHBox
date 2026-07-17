# 旧版能力审计与升级决策

## 结论

旧能力大多不是“准备删除”，而是在 9fce7870（remove legacy Container product stack）中随旧编辑器、服务和运行时整体移除。3.1 建立了更清晰的 Workflow Source、Authoring Projection、compiler/scheduler 和 capability/target 边界，但编辑器迁移尚未完整。

不回滚该提交，也不复制旧组件；必要能力必须在 3.1 唯一契约与执行路径上重做。

## 证据

旧版曾实现：

- useInlineMenu / inlineNodeCandidates：从输入或输出 pin 拖到空白处，按 exec/data、方向和类型筛候选，创建后自动连线。
- useGraphLayout / useElkLayout：六种对齐、水平/垂直分布、LR/TB 自动布局、实测尺寸、异步上下文保护和中心锚定。
- 多选、框选、批量移动、clipboard、节点/边/pin/多选菜单、吸附线、节点搜索、命令面板、问题面板。
- DebugStart/Step/Continue/Pause/Stop、从节点开始、当前节点/输出/变量/队列和副作用提示。
- 键鼠与轨迹录制、模板/Clip 资源、WaitTemplate、WaitTemplateGone、ClickTemplate。

当前 3.1：

- WorkflowEditorView 只有基础拖拽、连线、双击断线、添加、删除、编译和运行。
- EditorSession 的 connect 校验覆盖 TypeExpression、carrier、signal channel 和 instruction 入口。
- Run timeline 基于 journal，但没有暂停、单步、断点和 watches。
- nodes 是随 source/selection 重建的 computed 数组；手势实时位置在 Vue Flow 内部，只在 node-drag-stop 提交 event.node.position。这是位置漂移的最高概率竞态，但必须先仪器化确认。
- 当前 Debug 仍走 startRun，debugging 只是展示标志，因此命名具有误导性。

## 决策矩阵

| 能力 | 决策 | 3.1 适配 | 优先级 |
| --- | --- | --- | --- |
| 点击/连线不误移节点 | 修复 | 稳定交互投影、显式 drag handle、nodrag、一次手势一条命令 | P0 |
| 拖线到空白推荐节点并自动连接 | 恢复重做 | 共用连接兼容性服务，覆盖类型、carrier、channel、instruction/资源规则 | P0 |
| 连线 hover 可用状态和原因 | 恢复 | 提交前反馈，最终仍走权威校验 | P0 |
| 多选、框选、批量移动/Delete | 恢复 | Vue Flow selection 映射为原子 EditorSession 命令 | P0 |
| 对齐、分布、LR/TB 自动布局 | 恢复适配 | 实测尺寸、ELK、异步 revision token、单次 undo | P0 |
| clipboard、上下文菜单、吸附线 | 恢复精简版 | 统一 action registry 与明确手势边界 | P1 |
| 画布搜索、命令面板 | 恢复精简版 | 节点定位与动作搜索分开 | P1 |
| 结构化诊断、跳转/确定性修复 | 恢复 | compiler diagnostics 为唯一事实 | P1 |
| 运行轨迹和节点高亮 | 增强 | journal 派生，不改变执行 | P1 |
| 当前 Debug 名称 | 立即改名 | 真调试前称“运行并查看时间线” | P0 |
| 暂停、单步、继续、停止、断点 | 重新设计 | 同一 Program/scheduler/Owner/journal/capability | P2 |
| 从任意节点直接调试 | 不原样恢复 | 会跳过状态、target、资源和授权；除非 compiler 能证明前置闭包 | 不恢复 |
| WaitTemplate / WaitTemplateGone | 恢复重做 | exact target、BlobRef、timeout/poll/result/error | P2 |
| ClickTemplate | 恢复重做 | 匹配与点击同一 target/session 和坐标语义 | P2 |
| 模板安全缩略图、录制联动 | 恢复 | 受限内容 API，输出 3.1 source/草稿 | P2 |
| 注释框、reroute、subgraph 折叠、snippet | 延期 | 真实大图反馈后另定语义 | P3 |
| JS/yt 任意脚本入口 | 暂不恢复 | 扩大不可信代码和 capability 审计面 | 不恢复 |
| 旧 Container UI/第二运行时 | 明确删除 | 保持 3.1 唯一路径 | 不恢复 |

## 分阶段路线

### Stage 1：可靠而高效的图编辑

- [Slice 1：交互正确性](01-interaction-correctness.md)
- [Slice 2：类型感知连线](02-connection-authoring.md)
- [Slice 3：选择与布局](03-selection-layout.md)

三个 Slice 完成后统一运行前端聚合测试、task check、Windows build 和真实 GUI 手势 smoke。

### Stage 2：诚实的运行认知与真正调试

- [Slice 4：诊断与运行轨迹](04-diagnostics-run-trace.md)
- [Slice 5：唯一调度器上的真调试](05-true-debugger.md)

两个 Slice 完成后统一验收普通运行、暂停/恢复/停止、失败、资源释放和 Windows GUI。

### Stage 3：自动化领域便利能力

- [Slice 6：模板复合节点](06-template-convenience-nodes.md)
- [Slice 7：资源预览与录制联动](07-asset-authoring-integration.md)

用真实模板工作流端到端批量验收，再运行 task check、build 和真机 smoke。

## 跨阶段约束

- 版本只进入 version/manifest/binary metadata，不创建 nodes31 一类包名。
- 所有批量编辑是一个原子 undo/redo。
- 候选过滤、hover 和 connect 不维护三套规则。
- 调试不得绕过 compiler、target、capability、租约或副作用审计。
- 保存成功等原地动作不 toast；失败才使用统一 Nuxt UI 反馈。
- Slice 内只做继续开发必需的定向检查；阶段末批量验收。
