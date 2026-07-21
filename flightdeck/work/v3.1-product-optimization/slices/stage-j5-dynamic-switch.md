# J5 — 动态 Switch 稳定分支拓扑

## Goal

让 Switch 的 case 数量由 Source 实例配置决定，并使 Authoring、连线校验、Compiler Program 与 Runtime
共享同一个纯确定实例解析结果，不保留写死的八组假端口。

## Current

已完成实现、定向验证与完整 `task check`。静态 Contract 只声明 `value`、`in`、`default`、`failed` 和内容寻址的
Instance Resolver；`caseCount` 为新节点提供 3 个默认 case，可配置 1–32。缺少配置的旧 3.1 Source
仍解析为 8 个 case。减少数量会移除超范围 binding 和 edge。

## Next

按 build knowledge 执行 Windows WebView/真机 smoke。

## Verification

- `go test ./internal/nodeinstance ./internal/nodes ./internal/noderuntime ./internal/nodeauthoring ./internal/workflow/compiler ./internal/workflow/authoring -run 'TestSwitch|TestControlAndEventNodesHaveExplicitExecutionSemantics' -count=1`
- `go test ./internal/workflow/compiler -run TestCompileResolvesDynamicSwitchPortsIntoProgram -count=1`
- `pnpm -C frontend exec vitest run src/app/editor/EditorSession.test.ts`
- `pnpm -C frontend typecheck`
- `task check`

## Acceptance

- Catalog 中 Switch 的静态端口不再枚举 `case-1` 至 `case-8`。
- 同一个 resolver declaration 和 config 产生一致的有效数据输入、exec 输出和 Program 端口。
- Runtime 只匹配配置范围内的 case；旧 Source 未配置时保持八分支语义。
- 缩小 case 数量不会留下悬空 binding 或 edge。
