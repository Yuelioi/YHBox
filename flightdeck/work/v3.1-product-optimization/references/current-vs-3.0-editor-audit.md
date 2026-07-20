# 当前 3.1 与 3.0 reference 编辑器差异审计

## 审计范围

- 当前仓库的 WorkflowEditorView、WorkflowNode、WorkflowGraphCall、WorkflowInspector、EditorSession、DebugController、Node Authoring Projection 和视觉分析节点。
- 参考仓库 yotta-3.0-reference 的 ContainerEditorView、ContainerFlowNode、Subgraph virtual markers、PinLiteral、Inspector 和布局方式。
- 已有本地研究：user-device-test-1-research、user-device-test-2-design、visual-type-system-authoring、authoring-projection-3.1。
- 用户提供的当前真机截图和三次调试失败反馈。

本轮没有改产品代码，也没有把 3.0 reference 当成可直接复制的实现。

## 1. 状态表达

当前 frontend/src/app/editor/WorkflowNode.vue 的 nodeClasses 同时设置：

- selected：border-primary
- running：border-primary
- succeeded：border-success
- failed：border-error
- debugCurrent：warning ring

数组后面的运行状态会覆盖前面的选中边框。用户报告“成功后所有节点绿色，无法判断选中”与源码完全一致。节点头部已经有运行状态文字和色点，因此继续占用整个边框没有必要。

判定：这是状态体系问题，应一次性拆分 selection、execution、debug、validation，而不是只换一个颜色。

## 2. 框选与批量操作

当前 frontend/src/views/WorkflowEditorView.vue 保留了：

- selectedNodeIds
- WorkflowSelectionToolbar
- 复制、剪切、重复、删除
- 对齐、分布、自动布局
- 折叠为子图

但 VueFlow 只显式设置 delete-key-code=null，没有显式 selection-on-drag、selection key、multi selection key 或 pan-on-drag 契约。主能力仍在，直接手势已经漂移。旧 Topic Slice 03 的“框选已完成”不能代表当前版本。

判定：先定义鼠标与键盘交互契约，再实现和验收。批量工具无需重做底层。

## 3. 子图

当前 contracts/workflow/3.1/workflow-source.ts 已定义：

- Graph.inputs / outputs：typed data boundary
- Graph.entries：内部执行入口 endpoints
- Graph.exits：带 id 的 exec/error exits
- GraphCall：graphId、bindings、position

当前 WorkflowGraphCall.vue 只显示一个隐式 exec 输入 in、数据输入输出和命名 exits。EditorSession 的 collapseSelection 明确：

- 多个不同执行入口会报 selection has multiple execution entries
- 至少需要一个执行入口和一个 signal exit
- 数据边界会自动生成 typed ports
- 多个 exec/error 出口会生成命名 exit

3.0 reference 的可取之处是把 entry/output metadata 投影成 SubgraphInput 和 SubgraphOutput virtual markers，用户可以在画布上看见流程从哪里进入、从哪里离开。这些 marker 不需要恢复成 runtime 节点。

判定：底层 Source-native multigraph 健康，缺的是内部 authoring 投影。3.1 当前语义是一个调用入口和多个命名出口，不是任意多入口。

## 4. 调试

当前系统不是完全“不支持调试”：

- internal/application/application.go 有 StartDebugRun 和 ControlDebugRun。
- internal/workflow/compiler/debug.go 有并发安全 DebugController、paused/stepping/pause-pending/completed 状态与 checkpoint。
- EditorSession 有 startDebug、controlDebug、generation 合并。
- 前端有 DebuggerPanel、Step、Continue、Pause。

但用户的真实三节点工作流已经第三次在单步时卡住。这说明已有单元测试没有覆盖实际 Application worker、scheduler checkpoint、事件顺序、前端合并和 UI 控制的完整路径。

判定：调试目前是“实现存在，产品未验收”。必须做纵向诊断并设置去留门槛，不能继续写“完成待用户验收”。

## 5. 复杂节点与颜色分析

当前 Authoring Projection 的 FieldProjection 只有基础 control、constraints、default、title、description 等事实；PortProjection 和 TypeProjection 有 editorAdapter，但没有通用的：

- 分组与顺序
- 常用或高级优先级
- 单位
- 节点内联优先级
- 任务预设
- 帮助与下一步建议

当前 ColorRangeValueEditor 虽然已经是 editorAdapter，但默认仍直接展示 RGB/HSV 三通道的 minimum/maximum。与此同时，frontend ScreenPicker 已调用 backend.tools.extractColorRange，说明颜色采样和范围计算能力已经存在，只是没有接入节点创作。

判定：不是缺一个转换函数，也不是缺六个输入框。缺的是类型级创作控件与任务闭环。

## 6. 简易宏与精准录制

当前底层 internal/services/inputclip/model.go 已经定义原子事件：

- KeyDown / KeyUp
- MouseBtnDown / MouseBtnUp
- MouseMove
- RawDelta
- Scroll
- 同一时间戳下用 Seq 保序

canonicalizeStopResult 在 simple 模式下会过滤 MouseMove 和 RawDelta，但仍保留独立 KeyDown、KeyUp、MouseDown、MouseUp 和 Scroll，并且会在录制结束时补齐未释放输入。因此底层 hook、InputClip 和 held-input 安全模型不是这次问题的根源。

语义损失发生在 internal/services/recording/draft.go：

- analyzeRecording 从第一个键按下开始收集 activeKeys，直到所有键都释放，才生成一个 keys 动作。
- W 先按、D 后按、W 先松、D 后松会被压成同一个 keys=[W,D] 和一段总 duration。
- applyEditedActions 再把所有 KeyDown 放到动作起点、所有 KeyUp 放到动作终点。
- buildWorkflowDraft 又把简单动作降成 PressKeys、ClickPointer 和 Delay 线性节点。

因此交叠按键、跨动作 held state 和不同释放顺序无法 round-trip。当前 RecordingActionEditor 继续强化了这个错误模型：每行是 keys/click/scroll，并附带 delayUs 和 durationUs，而不是显式 KeyDown、KeyUp 和 Sleep。

产品入口也混在一起：

- AssetsView 和 WorkflowEditorView 都用同一个 simple/precise 下拉框选择录制模式。
- 两种录制共用保存弹窗和资源展示。
- simple 被当作可以编辑的 grouped action，precise 只显示提示，但仍然是同一工作流。

判定：

1. 简易录制应成为可编辑 Macro 资源，使用 KeyDown、KeyUp、MouseDown、MouseUp、Click、Scroll、Sleep 的有序 tagged union。
2. 精准录制继续使用保留原始时间、轨迹、RawDelta 和校准信息的 InputClip。
3. 两者从入口、资源列表、详情、编辑器、端口类型和回放节点分开。
4. 底层 hook、event codec 和回放 backend 可以复用，但这是私有实现细节。
5. simple 录制保存后不自动拆成画布节点；宏由独立回放节点执行并持有跨动作输入状态。

## 7. 3.0 reference 值得保留的部分

- 子图入口和多个出口的可见 virtual markers。
- 未连线主输入在节点卡片上的 PinLiteral 快速编辑。
- 节点类型、执行 pin、数据 pin 和错误出口的清晰视觉层级。
- 左节点目录、中央画布、右 Inspector、底部问题或日志面板的稳定职责。
- 自动布局、框选、多选和上下文批量操作的直接手势。
- 复杂参数通过专用 picker、调色板和区域选择器表达，而不是只给数字。

## 8. 3.0 reference 不恢复的部分

- Container store/service/runtime。
- 旧 Subgraph RPC、旧 debug manager 和旧 Node registry。
- 按 node kind 分发运行语义。
- 旧 schema、旧全局 singleton store 和旧虚拟节点持久化方式。
- 为每个旧节点重新创建一套不受 Node Contract 约束的 Inspector。

## 9. 最终判断

当前框架整体健康，但产品创作层不够完善，且过去的验收过度依赖局部测试。合理路线是：

1. 保持 3.1 Source、Compiler、Program、Admission 和 Scheduler。
2. 深化 Authoring Projection 与类型级 Editor Adapter。
3. 恢复旧版被验证过的交互模型，而不是恢复旧 runtime。
4. 把 Macro 与 Precise InputClip 作为两个产品领域，避免有损转换和入口混淆。
5. 所有能力必须通过可见入口、可理解创作、运行闭环和真机黄金路径后才能标记完成。
