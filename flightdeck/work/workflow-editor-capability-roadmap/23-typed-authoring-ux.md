# Slice 23 — Typed Authoring 体验

## Outcome

强类型主动帮助用户完成图，而不是只拒绝错误。

## Delivered

- EditorSession 对整张图做有预算的固定点实例专化；State slot 是声明权威，消费者不能反向扩大 StateRead 类型。
- 端口兼容支持 exact、assignable、generic-bind，使用 sealed Catalog projection 的 traits/relations。
- 从端口拖到空白时筛选真正兼容的节点；精确匹配优先，UI 标出精确、兼容、推导、无损/有损/可失败转换。
- Run 状态可搜索名称/类型；拖画布默认 Read，Alt 拖 Write，也可点击明确插入；新增按钮使用 primary token。
- State Read/Write 节点在插入时原子写入 slot config，并显示精确端口类型。
- 状态面板显示引用数、可定位第一个引用，并阻止直接删除仍被引用的状态；只允许具有合法默认值的 durable 类型。
- incompatible drop 诊断包含 source/target type，不再只显示无上下文的“不兼容”。
- 连接计划区分 direct/conversion/incompatible，带稳定 reason、源/目标类型和从 ConversionSpec 推导的候选；候选按 autoInsert、风险、total、cost 和 ID 稳定排序。
- 可转换连接会弹出策略选择；选择后插入真实转换节点与两条边，整个桥接是一次可撤销编辑。有损/parser 从不静默插入。
- durable 精确输出的拖线菜单提供 Promote to State；确认名称后原子创建同类型状态、插入已连接的 State Write，整个操作一次 Undo。

## Remaining

- 状态删除/改型显示图级引用与连线影响预览，并提供可撤销的安全批量重写。
- 大 Catalog/状态列表增加结果预算、取消和虚拟化。
- 为连接计划补充与 Compiler 的跨语言固定 fixture parity，防止 Projection 解释器与最终编译权威漂移。

## Acceptance journeys

Repeat.index 拖线首先看到 Integer 运算，同时能直连 Number 大于和 Log<Integer>；Number→Integer 显示四种策略且不静默插入；String→Number 可选择 integer parser 后安全提升或 number parser；retryCount(Integer) 可搜索、拖出精确 Read/Write并定位引用；安全转换插入和状态提升均保存为可见、可撤销的图编辑。
