# Workflow resource edit and safe exit

## Outcome / Question

工作流内既能创建、使用也能编辑键鼠宏；离开脏工作流时提供保存恢复路径，而不是只允许取消或破坏性放弃。

## Completion criterion

- 工作流资源面板的 Macro 项提供明确编辑按钮，复用同一 MacroService、MacroActionEditor 和 asset changed 刷新链路。
- 编辑宏使用共享 BaseModal，保存失败保留编辑内容并就地显示错误 toast；保存成功关闭并刷新当前资源列表。
- 脏工作流离开确认提供“取消 / 放弃修改 / 保存并退出”，保存失败阻止导航并保留工作区。
- 录制中和 pending 录制继续先执行各自的安全恢复确认，不被普通工作流保存分支绕过。

## Blocked by

- Slice 10 的工作流资源工作区和现有 Macro 编辑器。
- Slice 14 稳定画布职责后再增加资源编辑入口。

## Verification

- 组件/视图测试覆盖 Macro 编辑入口、保存与资源刷新。
- 路由守卫测试覆盖三种离开结果和保存失败。
- WebView 在工作流内打开已有宏、修改动作、保存并再次打开；再修改工作流并选择保存退出。

## Out of scope

- 重写 Macro 数据模型或回放 runtime。
- 在本 Slice 加入精准录制工作台编辑。
- 为所有确认框强制三按钮；只扩展共享确认器的可选 alternate action。

## Result

已完成。工作流资源 Dock 的 Macro 项可直接打开共享 MacroActionEditor，仍通过 MacroService 保存并触发 assets epoch 刷新。共享确认器增加可选 alternate action；脏工作流离开现在是取消、放弃修改、保存并退出三路，保存失败返回 false 并阻止导航。定向创作基础测试、TypeScript 类型检查与 i18n 检查通过。
