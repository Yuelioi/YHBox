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

Completed。移除没有暂停/单步能力的伪 Debug 入口，普通 Run 明确命名为“运行并查看时间线”。诊断按 error/warning/info 分组，显示 compiler 提供的 message、节点、字段/端口位置，只在 compiler 明确携带 fix 时展示修复声明；选择诊断会切换 graph、选中并居中节点。Run journal 的 node-attempt 事实派生节点状态与画布高亮，timeline 条目也能定位节点。诊断和 timeline 面板可关闭，并能从工具栏恢复，没有引入第二执行路径。

定向验证通过：vue-tsc、ESLint、i18n、oxfmt，以及 4 个相关测试文件 16 项；源码提交 9f06e9e8。按阶段验收原则未在本 Slice 重跑完整 task check，留到 Stage 2 末统一执行。
