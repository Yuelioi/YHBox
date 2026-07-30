# Runtime and lifecycle

`internal/localruntime.Runtime` 是 storage-backed 本地运行环境的 composition seam。Desktop 与 CLI
只向 `Open` 提交 profile 路径和 presentation adapter；它统一打开 storage/catalog/settings，
安装 AI provider，并配置 HTTP、应用与 automation target provider，构造 Blob、Script、Node Package 和 Workflow runtime，
并在 `Close` 中逆序释放全部所有权。presentation 不得重建其中任何一段安装或关闭路径。

`internal/appbootstrap` 的 concrete execution environment factory 从同一组配置事实一次性封存
Host Profile、Policy、Capability Provider snapshot 和 Configured Target snapshot。首次启动与 settings
热替换共用这条派生路径；新环境整体发布，旧 Run 持有旧对象的生命周期引用直至自然结束。Network、
Application 与 Automation Target 不进入 capability/admission/policy；每个 Run 从 target snapshot
创建直接调用 runtime，且没有 grant 或 TTL。

`internal/appruntime.Runtime` 管 desktop 后台资源的启动顺序。启动按依赖顺序执行；某一步失败时逆序
回滚已启动组件。关闭先释放输入/daemon/server，再关闭 `localruntime`，由后者拒绝新任务、取消运行并
flush settings/log transport；`Close` 幂等并聚合错误。

`internal/application.Application` 仍是所有 presentation 的统一命令入口。其 concrete
`sourceTransitions` 隐藏 Source patch、状态迁移检查、candidate prepare/commit 与 CAS；
concrete `runAdmission` 保证 Program 先持久化，并把 admission、Capability Provider snapshot、
Configured Target snapshot 与 generation 生命周期引用作为一个原子结果交给 concrete
`runCoordinator`。后者唯一拥有 queue、worker、Run Owner、debug session、provider 生命周期释放
和终态事件；调用方不能分别拼装这些步骤。

Workflow Wails adapter 只做 DTO 投影和单项命令转发。Library 搜索预算、筛选、facet、排序、分页，
以及批量 metadata/export/delete 的 duplicate、reference 和逐项结果策略由 concrete
`sourceLibrary` 统一拥有。

旧 `internal/services/container/runtime` 已删除。其 container-global vars、listener 子流程、kind dispatch 和私有 queue 语义不允许重建为 fallback；当前系统只通过 Application 执行 content-addressed Program，外部 listener 每次触发独立 Run。

取消通过 `context.Context` 传播。任何启动 goroutine 的组件都必须同时定义等待或关闭路径，不能依赖进程退出回收资源。
