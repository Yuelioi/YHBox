# ⚠ vue-flow 删除键无视修饰键
## Signature
- symptom: 按 Ctrl+Delete 想走自己的「彻底删除」逻辑(带确认框),结果 vue-flow 照常把节点引用删了、确认框根本没弹;底层定义没删。
- error_type: —
- where: vue-flow GraphView 的 `useKeyPress(deleteKeyCode)`(document 级 keydown 监听);frontend/src/composables/containerEditor/useEditorHotkeys.ts
- trigger: 在 vue-flow 画布上想做「修饰键 + Delete」的自定义删除手势,与 vue-flow 内置 delete-key-code 撞车

## 症状/复现

给「调用子图」节点加 Ctrl+Delete「彻底删除(连子图定义)」。期望:弹确认框 → 确认才删定义。实际:**节点引用被删、没弹确认框、底层子图还在**。换了两版键盘方案都没拿稳。

## 根因

`<VueFlow :delete-key-code="...">` 的删除键判定**只认键、不认修饰键**:
- vue-flow 内部 `useKeyPress(deleteKeyCode, ...)` 在 **`document`** 上挂 keydown 监听(`@vueuse` onKeyStroke,默认 bubble);命中即 `removeNodes(getSelectedNodes) + removeEdges(...)`。
- 你在 `window` 上(冒泡阶段)挂自己的处理 + `e.preventDefault()` —— **拦不住** vue-flow:`preventDefault` 不阻止其它监听器;且冒泡阶段 document 先于 window 触发,vue-flow 已经删了。
- 试图传**函数 filter**(`:delete-key-code="(e)=>!e.ctrlKey && ..."`)让 vue-flow 自己排除修饰键 —— 文档/类型上支持(`KeyFilter`),但本仓实测没稳住(疑 HMR 旧码 / 响应式取值时序),Ctrl+Delete 仍被删。
- 试图**捕获阶段** `window.addEventListener('keydown', fn, true) + stopImmediatePropagation` 抢在 vue-flow 前 —— 理论上 window 捕获先于 document,但实测**仍没稳住**(连第三版都 fail)。

一句话:跟 vue-flow 的键盘删除在同一根键(Delete)上抢,链路太脆,不值得。

## 修法

**别跟 vue-flow 抢键盘。带修饰键/特殊语义的删除手势改用非键盘 UI 入口。**
- 本案最终方案:右键节点 → **上下文菜单**条目「彻底删除(连子图/clip 定义)」(`NodeContextMenu.vue` specialItems 对 Subgraph/PlayClip 显示 → `useContextMenuRouter` onNodeMenuAction 'hard-delete' → 视图 `hardDeleteNodes([id])`:查引用 → `useConfirm` 确认框 → 删节点 + 删定义)。直接点击触发,无事件竞争,确认框稳弹。
- `:delete-key-code` 保持最简单的 `['Delete','Backspace']`,只管普通删除 + 删边。
- 通用结论:vue-flow 画布里,**普通 Delete 交给 vue-flow,任何「带修饰键 / 需确认 / 有副作用」的删除走右键菜单或按钮**,不要在键盘层跟它叠监听。

## Cases
- 2026-06-14 首次。录制产物/子图节点改进案中给 Subgraph/PlayClip 加「彻底删除连定义」,键盘方案(冒泡 filter → 捕获 stopImmediatePropagation)连 fail 两轮,最终改右键菜单解决。
