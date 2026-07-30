# Context

Yotta 是用户直接控制本机与已连接设备的自动化工具。HTTP 连接、应用和自动化目标由用户在当前设备上
配置；配置本身就是运行意图，不再叠加 Yotta 自己的 capability、授权、consent、grant、租约或 arm 状态。

## Product decisions

- 网络配置决定请求目标；运行时不再阻止私网地址、重定向或非 exact Origin。
- 应用配置决定可启动和停止的应用；运行时不再要求 application capability 或额外授权事实。
- 自动化 Target 是普通配置，负责选择 adapter 和目标参数；不再生成 capability requirement、Plan 或 Grant。
- Run 生命周期由工作流完成、用户取消、进程关闭或真实 adapter/设备错误结束，不由固定安全 TTL 结束。
- 配置格式、输入解析、资源回收、取消传播、持久化一致性和明确的设备不可用错误属于正确性，不属于授权步骤。
- 不以延长 TTL、自动续租、隐藏 consent 或 permissive grant 伪装完成；旧安全接口应从生产调用路径删除。

## Scope

本 Work 覆盖网络、应用与自动化目标的 Node Contract、Compiler/Admission 依赖、运行时装配、provider 调用、
设置与错误反馈。AI、文件、插件包信任和外部分发不因本 Work 自动改动，除非共享接口删除使其必须迁移。

## Final implementation

- `internal/targetruntime` 拥有三类 Configured Target 的 immutable snapshot 和 per-Run 直接调用生命周期。
- Node Contract/Authoring Projection 使用 `configuredTargets` 表达 slot、target kind 与 operation，不再
  为三类目标生成 Capability Requirement。
- execution environment 分别持有 Capability Provider snapshot 与 Configured Target snapshot；前者继续
  服务 AI、File、Blob、Stream 等资源，后者不经过 admission 或 Resource Broker。
- Network、Application、Automation 设置都是普通配置；热替换时旧 Run 只持有 provider 的生命周期引用，
  不存在授权租约或过期时间。
- 领域术语与架构入口见 [Repository context](../../../CONTEXT.md) 和
  [Architecture](../../../docs/architecture/README.md)。
