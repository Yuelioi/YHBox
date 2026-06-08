---
status: idea
summary: 节点编辑器 footgun backlog — ① exec 出口 Data 字段连不上(前提已变, 见正文需重评) ② DetectColor「范围」裸 JSON textarea 跟 HSV 结构化输入不一致
---

# 节点编辑器 footgun backlog

随手记的编辑器 UX bug，没承诺立刻干。知识层（"Data 字段不是 data-out pin、要走 GetSys"）已在 checklist [[2026-06-05-node-data-flow]]，这里只追"修前端"的待办。

---

## ① exec 出口的 Data 字段被渲染成可连输出口

`DetectColor` 的 `Center`/`Count` 这类 **exec 出口携带的 Data 字段**在画布上被画成**可连的输出口**，但直连会被 validator 拒（`INVALID_PIN out pin X 不存在`，且报错把 `Center` 显成小写）。画得出、连不上 = footgun。

修向：要么把这些 Data 字段渲染成**不可连**；要么连线时直接报错并指向正解 `GetSys($sys.lastColor.center)`。根因/可连判定锚点见 [[2026-06-05-node-data-flow]] 的「源码锚点」。

> ⚠️ **2026-06-07 前提已变，需重评**：(a) remove-sys 已删 `$sys`/`GetSys`——「正解 GetSys」不复存在；(b) error-model 给「exec 出口下的 Data 字段」加了 data-out 识别 + 运行时 exec-data bridge（`IsExecOutputDataField` + `applyExecDataEdges`，见 [docs/error-model.md](../docs/error-model.md)「消费 Error/Code」），**Fail 出口的 Error/Code 现在能连**（约束：data 线须跟父 exec 边并行）。DetectColor 的 Center/Count 在 remove-sys 后是否还是 exec-出口 Data 字段、是否已纳入该 bridge，未重新核实——动它前先看当前 DetectColor 输出结构 + 该 bridge 是否覆盖。

## ② DetectColor「范围」是裸 JSON textarea

`DetectColor` 的「范围」输入是裸 JSON textarea，跟 `DetectColorHSV` 的结构化 HSV 输入不一致（值是数组 vs 对象）。套同一结构化 widget 要小改 `parseRange6`。
