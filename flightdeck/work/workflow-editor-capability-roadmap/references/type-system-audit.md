# 3.1 类型系统审计

## 用户症状

- Repeat.index 是 Integer，GreaterThan 只收 Number；无法连线，也没有合理整数候选。
- Run 状态声明类型，但 StateRead/Write 画布实例仍是 T；状态不能搜索、拖出或导航引用。
- 模板匹配、二维码、颜色块、文件元数据等结构结果只有整体类型，没有字段消费入口。
- 删除 desktop application 留下 target 引用，Settings validator 拒绝整个更新。

## 根因

1. datatype.Assignable 对具体 ref 只接受完整 TypeRef 相等，没有显式关系。
2. TypeExpression 允许 constraints，compiler/type_solver.go 却拒绝所有非空 constraint。
3. solver 是 compiler 私有单向绑定器，没有公共实例求解、LUB 或稳定诊断。
4. 前端 connectionCompatibility.ts 复制简化规则，并将 variable 一律判 false。
5. State 节点靠 Compiler 从 slot 绑定 T，Authoring 只有基础 NodeProjection，没有实例 projection。
6. Data Type semantic 只有 schema/representation；Projection 没有 relation、trait、结构字段或 conversion graph。
7. 转换是无机器语义的普通节点，没有 total/lossless/fallible/cost/autoInsert。
8. Integer/Number 节点不闭合：循环和长度产出 Integer，比较/算术主要消费 Number。
9. InstanceResolver 只有声明和 seal，没有 registry、调用、Projection 或 Compiler 链路。
10. Catalog 没有 Type × Capability 门禁，“类型存在但无法消费”仍可通过。

## 决策

- 保留 TypeRef(typeId + semanticDigest) 的名义身份；不从 schema primitive 猜领域兼容。
- 区分 exact、assignable、generic-bind、lossless-convert、lossy-or-fallible、narrow-or-assert。
- Integer 与 Number 都满足 Numeric/Ordered；混合运算采用显式安全提升计划。Runtime 不做隐藏 coercion；改变表示必须为可见节点。
- duration/key-code 等领域类型不因底层表示而成为基础 Integer/String。
- 前端只消费 sealed Catalog 生成的 relation、solver result 和 ConnectionPlan。
- 每个公开类型必须满足适用的 literal/state/observe/equality/operation/collection/conversion/serialization/debug 能力或有 waiver。

## 研究依据

官方资料共同表明严格类型体验依赖类型推导、重载/泛型、可见转换、上下文候选、变量拖拽和原因诊断，而不是只拒绝错误连接。完整来源和本地应用见 docs/research/visual-type-system-authoring.md。

## 回归入口

- internal/services/settings_test.go：删除 application 同时删除依赖 target。
- frontend/src/app/editor/EditorSession.test.ts：StateRead 配置后实例投影解析声明类型。
- frontend/src/app/editor/connectionCompatibility.test.ts：改为 Relation/ConnectionPlan 测试，不假设孤立 converter。
