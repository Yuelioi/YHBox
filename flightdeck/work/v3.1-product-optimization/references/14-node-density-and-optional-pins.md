# Node density and optional pins

## Outcome / Question

让画布节点回归“识别、连接和查看状态”的紧凑职责；复杂节点默认只展示必要、主要或已连接输入，可选输入按需展开，common 调优参数主要留在 Inspector。

## Completion criterion

- 普通节点使用稳定固定宽度，长 pin 名称不会靠 intrinsic width 撑宽卡片。
- signal pins、required inputs、primary inputs 和已连接 inputs 始终可见；其余 default/optional inputs 进入可展开区域，并显示隐藏数量。
- 仅当节点恰好存在一个适合画布的轻量 inline candidate 时显示单行编辑器；多个 common candidate 不择一误导用户，也不同时展开。
- 点击模板默认不显示 timeout、poll interval、settle duration、button、hold duration 等可选输入和三个内联时长；展开后所有 pins 仍可连接，Inspector 仍显示完整参数。
- 连接到原本隐藏的输入后，该 pin 自动可见且边不丢失。

## Blocked by

- Slice 07 的 Authoring Surface 与 Slice 11 的画布滚轮/尺寸验收。
- 不修改 Node Contract 运行语义；只调整 Authoring Projection 的画布投影与本地显示状态。

## Verification

- authoringSurface 单元测试覆盖零、单个和多个 inline candidate。
- WorkflowNode 组件测试覆盖固定宽度、隐藏计数、展开和 connected pin 自动可见。
- 真实 WebView 插入点击模板，按 Vue Flow zoom 归一验证宽高，并验证收起/展开后 wheel 与连线行为。

## Out of scope

- 隐藏 required、primary、connected pin。
- 新增全局“基础/专业”能力开关。
- 改变 Inspector 参数分组或 runtime 默认值。

## Result

已完成。WorkflowNode 使用 260px 固定宽度；signal、required、primary、connected inputs 保持可见，其余输入显示隐藏数量并可展开。Authoring Surface 只在恰好一个轻量候选时内联，因此点击模板的 common 时序参数回到 Inspector。定向 authoringSurface 测试、创作基础测试、TypeScript 类型检查与 i18n 检查通过。
