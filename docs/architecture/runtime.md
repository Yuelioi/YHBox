# Runtime and lifecycle

`internal/appruntime.Runtime` 是后台资源的唯一生命周期 owner。启动按依赖顺序执行；某一步失败时逆序回滚已启动组件。关闭先拒绝新任务并取消运行，再释放输入/daemon/server，最后 flush 日志；`Close` 幂等并聚合错误。

旧 `internal/services/container/runtime` 已删除。其 container-global vars、listener 子流程、kind dispatch 和私有 queue 语义不允许重建为 fallback；3.1 只通过 Application 执行 content-addressed Program，外部 listener 每次触发独立 admitted Run。

取消通过 `context.Context` 传播。任何启动 goroutine 的组件都必须同时定义等待或关闭路径，不能依赖进程退出回收资源。
