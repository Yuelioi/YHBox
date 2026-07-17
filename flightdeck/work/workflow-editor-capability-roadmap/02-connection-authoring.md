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

Slice 1。

## Verification

定向验证候选矩阵和原子 undo；Stage 1 末批量验收。

## Out of scope

跨 graph、隐式类型转换、动态 schema 和脚本节点。

## Result

Planned。保留旧交互价值，但不复制旧类型启发式。
