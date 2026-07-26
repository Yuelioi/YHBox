# J2 — 运行停滞可解释性

## Goal

让正在运行或等待外部条件的 Workflow 明确显示当前 Node Attempt、等待原因、已经过时间、预期超时
与最近终态路由，使用户不用猜测“卡在哪里”。

## Current

已完成。模板等待节点现在写入 `waiting`、`matched`、`timeout` Status Event；运行时间线会自动打开并
显示当前节点、attempt、等待原因、已用时间与 timeout 预算，画布同步显示等待态。timeout 路由未连接时
会明确提示该分支未处理。

## Next

进入 J3，修复节点发现、节点右键菜单定位和大规模下拉查找。

## Verification

- `pnpm -C frontend exec vitest run src/app/editor/runTrace.test.ts src/app/editor/WorkflowRuntimePanels.spec.ts`
- `pnpm -C frontend typecheck`
- `pnpm -C frontend i18n:check`
- `go test ./internal/nodes ./internal/noderuntime -run 'TestAutomationTemplateContractsUseExplicitBlobAndExactTargetCapabilities|TestClickTemplateCapturesMatchesAndClicksTheSameExactTarget|TestEmitTemplateMatchStatusDistinguishesMatchAndTimeout' -count=1`

## Acceptance

- Run 进行中时，画布和运行工作台能定位当前节点与 attempt 状态。
- 外部等待节点至少显示等待对象、已经过时间以及配置的 timeout；终态显示实际选择的分支。
- timeout/failed 分支未连接时明确提示运行会在该路由终止，而不是继续显示全局运行中。
- 运行信息来自 Run Record、Node Attempt 或 Status Event，不由前端计时器伪造执行状态。
- 定向 Go/前端测试通过；敏感输入输出仍遵守脱敏边界。
