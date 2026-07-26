# K1 — 第二批真机反馈闭环

## Goal

修复 `FD-11` 至 `FD-18`，让节点发现、运行状态、动态端口、工作台布局和临时调试草稿在真实宿主中
保持一致且可解释。

## Current

功能与定向测试已完成。Catalog 图标全部可解析；诊断进入底部工作台；两侧栏可折叠和缩放；Run State
支持所有可安全初始化类型及组合键；Switch case 标题唯一；通用节点菜单不再暴露模板动作；选择节点
恢复 Inspector；Compiler 不再让缺少必填输入的悬空 pull 草稿阻塞事件主链运行。全量门禁捕获的独立
pure-data 执行根回归已修复；质量门禁已分为日常增量和 CI/发布 full 两层。

## Next

执行 Windows WebView/真机 smoke；CI/发布再运行 `task check:full`。

## Verification

- `pnpm -C frontend exec vitest run src/test/tablerIconReferences.spec.ts src/app/editor/EditorSession.test.ts src/app/editor/WorkflowRuntimePanels.spec.ts src/views/WorkflowAuthoringFoundations.spec.ts src/app/editor/stateVariableTypes.test.ts`
- `go test ./internal/workflow/compiler ./internal/nodes ./internal/nodeauthoring ./internal/noderuntime -count=1`
- `pnpm -C frontend typecheck`
- `task check`（按变更范围）

增量结果：Go 23 个受影响/反向依赖包及 vet 通过；前端 format、lint、typecheck、i18n 和 67 files / 276
tests 通过。完整门禁曾捕获 pure-data 执行根回归，修复后原失败的 compiler/application/service fixture
均通过；遵循新分层不在普通修复中重复 full。

## Acceptance

- 用户列出的八个节点均显示 Tabler 风格图标。
- 诊断与日志、时间线、调试共享底部工作台；左右侧栏可隐藏和调整宽度。
- Run State 组合键可接入对应节点，复杂 durable 类型能以合法默认值创建。
- Switch 的 case 输入和输出数量、标签一致。
- 普通节点右键无视觉模板动作；点击或定位节点立即显示 Inspector。
- 事件主链可在旁边存在未连接草稿节点时编译和运行，草稿节点不进入 Program。
