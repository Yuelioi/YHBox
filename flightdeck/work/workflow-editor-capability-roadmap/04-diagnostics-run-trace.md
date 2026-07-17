# Slice 4：结构化诊断与运行轨迹

## Outcome / Question

明确区分编译问题、普通运行轨迹和真正调试，并能从问题定位节点。

## Completion criterion

- 真调试交付前把 Debug 改为“运行并查看时间线”。
- 诊断按 error/warning 分组，可定位节点/端口；只展示 compiler 声明的确定修复。
- 从 journal 派生节点运行状态和画布高亮，不改变执行。
- 保存成功原地反馈；失败用 Nuxt UI，不用浏览器 alert 或成功 toast。
- run/compile 状态可恢复、可关闭。

## Blocked by

Stage 1。

## Verification

定向验证诊断映射、跳转和 timeline；Stage 2 末批量验收。

## Out of scope

暂停、单步、断点、watches。

## Result

Planned。timeline 保留，但不再冒充 Debug。
