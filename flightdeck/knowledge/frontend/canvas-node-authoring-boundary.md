---
kind: trap
summary: "画布节点只能内联轻量编辑器；复合 typed editor 留在 Inspector，几何与滚轮必须用真实 viewport/WebView 契约验收。"
activation: symptom
read_when: "复杂节点异常膨胀、节点上滚轮不能缩放，或准备修改 Authoring Surface inlinePriority/editorAdapter 与 WebView 画布 smoke 时"
recheck_when: "Authoring Surface 内联契约、Vue Flow 版本或 WebView 输入栈变化时"
---

# 画布节点 Authoring 边界

## 陷阱

`inlinePriority` 只表达业务重要性，不代表控件适合放进画布节点。ColorRange、Point、Region、Asset、JSON 等复合 editor 即使优先级高，也会把完整表单、滚动和拾取交互带进节点，造成节点尺寸失控并争夺 wheel。画布节点只内联轻量、单行、无独立滚动面的 adapter，而且一个节点必须恰好只有一个合格候选才内联；多个 common 候选同时出现时全部退回 Inspector，不能擅自取前三个或取最高项。

## 验收方式

- 测节点布局尺寸时使用 `offsetWidth` / `offsetHeight` 或按 Vue Flow viewport zoom 归一后的尺寸；直接用 `getBoundingClientRect()` 会把相机缩放误判成节点变大。
- 浏览器脚本构造的 `WheelEvent` 不是 d3/Vue Flow 在 WebView 中的可信输入证据。使用 CDP `Input.dispatchMouseEvent` 的 `mouseWheel`，并断言 viewport zoom 实际改变。
- wheel 缩放要保持光标下的图坐标不漂移，并对最小/最大 zoom 做单元测试。
- 真机 fixture 应使用最高复杂度节点（当前为 Analyze Color），同时断言节点尺寸、复合 inline adapter 数量和后续选择/连接/保存旅程。wheel 必须在清空全部选择后分别覆盖空白画布、未选中节点和已选中节点，不能让“新插入节点自动选中”制造假绿。

## 当前契约

允许内联的 adapter 为 `duration`、`key-chord`、`number`、`select`、`text`、`toggle`。新增 adapter 若包含多字段布局、画布拾取、资源浏览或独立滚动区域，默认进入 Inspector，不得仅凭 `inlinePriority` 投影到节点卡片。

节点卡片应使用稳定固定宽度。signal、required、primary 和已连接输入始终显示；其余 default/optional 输入默认收起并显示隐藏数量，用户展开后仍可正常连接。这样“画布负责识别与连线、Inspector 负责完整参数”的边界不依赖具体节点数量。
