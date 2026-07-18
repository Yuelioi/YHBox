---
kind: trap
summary: "历史 3.0 ContainerFlowNode/PinLiteral 结构化输入陷阱；现行 3.1 仅在旧 UI 取证时读取。"
activation: symptom
read_when: "仅在审查 3.0 ContainerFlowNode/PinLiteral/StructuredInput 旧实现或旧截图行为时"
---
# ⚠ 画布内联 literal 只认 scalar; 结构化类型(point/geometry)手填走 Inspector

> 历史知识：相关组件和旧 Pin 类型已删除。3.1 控件由 Data Type/Authoring Projection 与 Editor Adapter 决定，不能据本文修改现行 UI。
**Date**: 2026-06-24 (Phase 4 Point 手填实测踩坑)

前端 pin 字面量编辑有**两套渲染路径**, 别混:

1. **画布内联**(`ContainerFlowNode.vue` → `showInlineLiteral` → `inlineLiteralPins` → `PinLiteral.vue`):只认 scalar —— `dropdown`(有 options) / `list`(wire-only 提示) / `number` / `bool` / **else = `UInput` 文本**。结构化值(Point/Geometry/object)落 else → `String({x,y})` = `"[object Object]"`。所以 `inlineLiteralPins` 的 filter **排除 point/geometry/code**(只留 scalar)。
2. **Inspector**(`NodeInspector.vue` → `dataInLiterals` → 有 `schema` 走 `StructuredInput.vue`):渲染结构化 widget —— `schema.widget==='geometry'`→GeometryWidget, `'point'`→PointWidget, object/tuple→递归。`unconnectedDataInPins`(pinLiterals.ts)返回**所有**未连线 data-in(含结构化, 不 filter);画布额外 filter, Inspector 不 filter。

**给结构化 pin 类型加手填的正确套路**(以 Point 为例):
- 后端: pin 加 `Schema`(`node.PointSchema()` = `&FieldSchema{Type:"object", Widget:"point"}`, 仿 GeometrySchema)。
- 前端: `StructuredInput.vue` 加 `schema.widget==='point'` 分支 → 新 Widget(PointWidget 仿 GeometryWidget, 显示 0-100% / 存 0-1 ratio)。
- **别动画布 `inlineLiteralPins`** —— 画布 PinLiteral 不渲染结构化, 放进去必 `[object Object]`。手填走 Inspector 即可(同 Geometry ROI 一贯做法)。
- 后端值解析**已支持, 零改**: `data_pull.go` `coerceToType(v,"Point")` 把手填字面量 `map{x,y}` / `[x,y]` / `expr.Point` 转 `node.Point`(`in.Point` 仅类型断言, 不用动; 有 `TestCoerceToType_Point` 背书)。

**踩过的坑**: 给 Point 加手填时, 误把 point 放进画布 `inlineLiteralPins` + 顺带删了 geometry 排除 → 画布 point/geometry 全 `[object Object]`(geometry 回归)。修复 = `inlineLiteralPins` filter 回滚到 scalar-only(`p.type!=='point' && schema.widget!=='geometry' && widgetKind!=='code'`)。
