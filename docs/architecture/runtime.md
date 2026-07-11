# Runtime and lifecycle

`internal/appruntime.Runtime` 是后台资源的唯一生命周期 owner。启动按依赖顺序执行；某一步失败时逆序回滚已启动组件。关闭先拒绝新任务并取消运行，再释放输入/daemon/server，最后 flush 日志；`Close` 幂等并聚合错误。

Container runtime 在构造期编译主图和子图，捕获一个 node registry generation，并建立私有零分配 execution table。每个 runner 持独立 dispatch state；listener 子流程共享不可变编译产物与 container-global vars，但拥有独立 frame、queue 和 service bundle。

取消通过 `context.Context` 传播。任何启动 goroutine 的组件都必须同时定义等待或关闭路径，不能依赖进程退出回收资源。

