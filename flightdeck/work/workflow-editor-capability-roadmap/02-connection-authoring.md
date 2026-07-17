# Slice 2：类型感知的连线创作

## Outcome / Question

从输入/输出 handle 拖到空白处时展示兼容节点，选择后可靠创建并自动连线；拖过端口时提前解释能否连接。

## Completion criterion

- 兼容性覆盖 data/exec/error、方向、TypeExpression、carrier、instruction 与必要资源约束。
- 候选、hover、最终 connect 共用规则/预览。
- 单一端口自动连，多端口显式选择；无候选说明原因并可查看全部。
- 创建+连线是一个 undo，失败无孤儿节点。
- 输入反拖、输出正拖、单输入数据边替换均有明确行为。
- 原地显示连接失败，系统失败才 toast。

## Blocked by

Slice 1。已解除。

## Verification

已运行连接兼容性、EditorSession、位置投影和 UI 接线定向测试：4 个文件、18 项通过；pnpm typecheck、定向 oxlint 与 pnpm i18n:check 全部通过。Stage 1 末统一做真实 GUI 与完整门禁。

## Out of scope

跨 graph、隐式类型转换、动态 schema 和脚本节点。

## Result

Completed。新增 projectedConnectionCompatibility 作为候选过滤、Vue Flow isValidConnection 和最终 EditorSession connect 的共同规则，覆盖名义类型、union/list、carrier、resource lease operation 收窄、exec/error channel 与 instruction 入口。拖线落空打开 WorkflowConnectionMenu；候选按具体端口列出，选择后由 insert-connected-node 原子命令新增并连接，undo/redo 一次完成且保存时展开为正式 add-node + connect patch。“显示全部”保留逃生口但明确只添加不兼容节点。同步修正 error edge 的 targetHandle channel。
