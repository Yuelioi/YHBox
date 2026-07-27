# V4-O Program / Adapter / Compiler / Executor 边界

## Goal

让 Program artifact、Source compiler、node adapter host ABI 与唯一 production Executor 拥有准确依赖
方向，消除 adapter 实现为了 ABI 反向依赖整个 `workflow/compiler` 的 seam penetration。

## Status

Finished

## Completed

- 仅由测试调用的 pure-data Interpreter 移入 `_test.go`，不再进入 production build。
- 新增 `internal/nodeadapter`，拥有 Adapter、Invocation、Result、Failure、SignalTrigger、
  AdapterAction 与 invocation-scoped StateBinding ABI。
- `internal/noderuntime` 和 `internal/pluginhost` 的 production 文件全部改为依赖 `nodeadapter`；
  新增架构测试禁止它们重新导入 `workflow/compiler`。
- `workflow/compiler` 只消费该 ABI；测试别名留在 `_test.go`，没有恢复 production re-export。
- `ProgramSnapshot` 保持唯一 immutable artifact/interface，durable opener、Source compiler 与
  production Executor 继续在同一深 Module 内共享私有 graph/input plan；不为了拆包公开 mutable
  Program document 或复制一套 execution view。

## Verification

- `go test ./internal/nodeadapter ./internal/workflow/compiler ./internal/noderuntime ./internal/pluginhost ./internal/appbootstrap ./internal/architecture -count=1`
- 六个包通过；中间切片不运行 `task check`。

## Next

继续 [Application 深化](v4-p-application-modules.md)。
