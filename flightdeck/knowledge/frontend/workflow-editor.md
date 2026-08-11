# Workflow editor development

## Ownership map

| Concern | Owner |
| --- | --- |
| 页面组合、Vue Flow wiring、dock/panel | `frontend/src/views/WorkflowEditorView.vue` |
| 当前 Source、revision、selection domain、undo/save commands | `frontend/src/app/editor/EditorSession.ts` |
| 创建 session 与 transport | `createEditorSession.ts`、`frontend/src/app/transport/workflow.ts` |
| layout、selection、run、resource 等动作 | `Editor*Controller.ts` |
| Source node → Vue Flow node 投影 | `workflowFlowProjection.ts` |
| canvas gesture props | `workflowCanvasInteraction.ts` |
| node/inspector/value editor | `frontend/src/app/editor/WorkflowNode.vue`、`WorkflowInspector.vue`、`WorkflowValueEditor.vue` |

View 负责组合，Controller 负责一个用户动作，EditorSession 负责可持久的 domain state 与 command。不要把
revision CAS、保存、运行或资源副作用重新塞进 Vue component watcher。

## Vue Flow authority

- Workflow Source/EditorSession 是持久事实；只有 drag-stop、typed patch 等明确 command 才写回 position、
  config、edge 或 selection domain。
- Vue Flow store 拥有瞬时 drag position、marquee 和 selected state。外部 `flowNodes` projection 不携带
  `selected`，也不能在每次 Source/selection refresh 用旧 position 覆盖内部 live position。
- 拖拽中的位置来自 `event.node.position` 或 live gesture overlay；不能从外部 computed nodes 回读。
- 当前 graph 共用一台 Vue Flow camera；切 graph 时保存/恢复 graph viewport，不能为嵌套层级创建第二个
  store/camera 或把 viewport 混进 Workflow semantic digest。
- node header 是 drag handle；port、input、button、scrollable editor 标记 `nodrag`/`nowheel`，避免点击被
  1px drag threshold 吞掉。

## Canvas gestures

当前桌面交互合同是：

- 空白画布左键拖拽：框选；不要求额外修饰键。
- `Ctrl` + click/框选：追加或切换多选。
- `Space` + 左键拖拽：平移画布。
- 中键或右键拖拽：平移画布。
- wheel/trackpad：按 Vue Flow 配置缩放；交互控件内部 wheel 不应误操作 camera。

对应配置是 `selectionKeyCode: true`、`multiSelectionKeyCode: 'Control'`、
`panActivationKeyCode: 'Space'`、`panOnDrag: [0, 1, 2]`。这些 prop 在 Vue Flow 内部共同决定优先级；修改时
必须挂载真实 `VueFlow` 验证 `.vue-flow__selection` 和 camera，不只测常量对象。Delete/Backspace 的业务
删除由 editor action 处理，必须显式区分输入焦点和 modifier，不能只依赖 Vue Flow 的 key-code prop。

## Typed value editors

复杂字段按 Data Type 的 Authoring Projection 选择具名 editor adapter，页面不按 node type ID 写大 switch。
节点卡片只内联轻量、单行、无独立滚动面的值；Point、Region、Asset、JSON 等完整编辑留在 Inspector。

KeyChord 是 key-code list，`ALT`、`CTRL`、`SHIFT` 等 modifier-only chord 是合法值。组合键在非 modifier
keydown 时提交；单独 modifier 必须等 keyup 才能判定，避免按下 Alt 后过早提交并破坏 Alt+A。相关边界由
`keyChord.test.ts` 和 `KeyChordValueEditor.test.ts` 覆盖。

## Verification

- domain/session/controller 用 Vitest 测 revision、命令和错误路径。
- Vue Flow 同步、选择、drag 和 camera 必须挂载真实组件测试；框架行为不确定时读锁定版本源码。
- 关键用户旅程用 CLI Playwright/Wails CDP 发真实 `keyDown → mouse → keyUp`，再看截图；在 mouse event 上
  只写 modifier 位不能证明 WebView 收到了真实多选手势。
- `task webview:smoke` 覆盖列表、恢复面、编辑器、资源和 schedule；修改 launcher 再运行 full 旅程。
