# Slice 23 — Typed Authoring 体验

## Outcome

强类型主动帮助用户完成图，而不是只拒绝错误。

## Delivered

- EditorSession 图级固定点实例专化；State slot 是声明权威。
- 端口兼容支持 exact、assignable、generic-bind，并从 sealed Projection 消费 traits/relations。
- 拖线候选按精确、兼容、推导和转换风险排序。
- Run 状态支持搜索、Read/Write 拖放、引用计数和定位；只允许有合法默认值的 durable 类型。
- 连接计划区分 direct/conversion/incompatible，带稳定 reason、源/目标类型和 ConversionSpec 候选。
- 可转换连接由用户选择策略后插入真实转换节点与两条边，整个桥接一次 Undo；有损/parser 从不静默插入。
- durable 精确输出支持 Promote to State：创建同类型状态、插入已连接 State Write，整个操作一次 Undo。
- Workflow Authoring Patch 支持 update-state-variable。无引用状态可改型并重建默认值；有引用状态由 UI 和后端双重阻止，避免产生隐式断线或无效图。
- 状态与连接候选列表加入分段结果预算，搜索或模式切换会重置预算。

## Remaining

- 引用状态改型显示跨图 Read/Write 和连线影响，提供显式、可撤销且能证明安全的迁移；不支持的路径给出逐项修复入口。
- 为 Projection 连接计划补充与 Compiler 的跨语言固定 fixture parity。
- 若实际目录规模超过分段渲染预算，再替换为测量过的虚拟列表，不预先引入复杂依赖。

## Acceptance journeys

Repeat.index 优先看到 Integer 运算，能直连 Number 大于和 Log<Integer>；Number→Integer 显示四种策略且不静默插入；String→Number 可选择 integer parser 后安全提升或 number parser；retryCount(Integer) 可搜索、拖出精确 Read/Write并定位引用；安全转换、状态提升和无引用改型均为可见、可撤销编辑。
