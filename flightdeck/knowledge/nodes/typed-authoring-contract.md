---
kind: checklist
summary: "3.1 强类型创作统一契约：名义类型、显式关系、非语义 Authoring Surface、可见转换和 Catalog 驱动候选形成闭环"
activation: action
read_when: "修改 Data Type、TypeExpression、泛型节点、转换节点、端口连线、Run 状态类型、拖线候选或结构化值字段时"
recheck_when: "类型关系、转换自动插入、constraint registry、节点能力覆盖或 Authoring Projection 改变时"
---
# Typed authoring contract

## 不变量

- TypeRef(typeId + semanticDigest) 是精确名义身份；不得凭 schema primitive、颜色、标题或 ID 后缀推断领域兼容。
- 关系区分 exact、assignable、generic-bind、lossless-convert、lossy-or-fallible、narrow-or-assert；调用方不得压成无原因 boolean。
- Compiler 是最终权威，Projection 来自同一 sealed Catalog；前端不得维护独立关系表、constraint 解释器或转换图。
- 泛型 T 按节点实例 scope 求解；constraint 必须来自封装 registry 并被执行。
- Runtime 不隐藏 coercion。改变表示的转换必须是保存图中的可见节点；自动插入只允许唯一、总函数、确定、无副作用、无损、无 capability 的单步转换。
- 解析、舍入、截断、单位/坐标/编码变化、carrier 变化、narrowing 和 assertion 永不静默自动插入。
- 领域类型不因表示是 integer/string 就自动成为基础类型；列表默认不变。
- StateRead/Write 由 slot declaration 专化，连接不能反向改变声明。
- 新 TypeRef 必须通过适用的 literal/state/observe/equality/operation/collection/conversion/serialization/debug 覆盖或 waiver。
- Node default/config builder 每次返回新的深拷贝值；实例插入 Source command 后由该 revision 独占，禁止多个节点或 revision 共享可变默认对象。
- group、order、importance、unit、inlinePriority、preset、help 与 editorAdapter 只属于非语义 Authoring Projection；不得改变 type、binding、carrier、capability 或 execution，也不得进入 semantic digest。
- Inspector 与节点卡片必须消费同一 Authoring Surface 投影；页面不得按 nodeTypeId/kind 分支复杂字段。Point、Region、Duration、KeyChord、Asset、Target 等差异由类型级 Editor Adapter 隐藏，未知 adapter 安全退回通用 typed editor。
- 节点卡片只可内联最多三个高优先级、未连线且具 inline-json 表示的输入；端口仍保留连接能力。复杂编辑器按需加载，不得为了改善创作体验突破编辑器初始 bundle 预算。
- Point/Region 的 ratio/px 切换是用户发起的显式坐标转换，必须依赖当前有效 target 尺寸换算；无 target 时不得只改 unit 字段造成静默语义变化。ScreenPicker 只返回 typed value，不能持有 target authority 或执行工作流。

## 修改顺序

Data Type/Node Contract semantic → Type System/Compiler → Projection/fixture → frontend client/instance projection → node runtime → parity matrix。阶段末才跑完整门禁。

## 用户旅程基线

Repeat.index 可直接比较、计算、观察和写 Integer state；拖线候选按直接/泛型/无损/显式排序；安全转换可见可撤销；Run 状态可搜索并拖出 typed Read/Write。
