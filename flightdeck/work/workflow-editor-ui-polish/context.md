# Workflow Editor UI Polish context

## What matters

编辑器反馈层级应与动作范围匹配：原地动作在原控件附近反馈，运行与调试使用自己的状态工作台，
只有错误和跨上下文结果使用 toast。确认流程必须是可测试的应用 UI，不调用浏览器原生 dialog。

## Decisions

- Vue Flow chrome 使用产品 semantic tokens，而不是硬编码浅色。
- Handle 的视觉和命中区域都必须位于节点边界且不覆盖标签。
- 保存/编译成功不制造 toast 洪水，失败保持清晰可恢复。

## Terms

- **Inline feedback:** 在触发控件或当前任务区域展示的短时结果。
- **Shared confirmation:** 由应用统一管理、可异步等待且可自动化测试的确认界面。
