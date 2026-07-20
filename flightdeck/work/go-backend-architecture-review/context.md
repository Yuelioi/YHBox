# Go Backend Architecture Review context

## What matters

本 Work 只证明仓库内工程边界已经加固，不代表公开发布治理已经完成。LICENSE 仍是
source-available；远端 CI、签名、公开仓库和 owner settings 必须由具备权限的维护者单独推进。

## Decisions

- GUI composition、应用服务和持久化 owner 保持深模块边界，入口不持有业务生命周期。
- 回滚与持久化失败必须立即隔离缓存，不允许内存状态伪装成 durable success。
- 跨平台结论区分 portable-core compile 与真实宿主 GUI/installer smoke。

## Terms

- **Portable core:** 不依赖具体 GUI 宿主、可在目标 OS 编译或测试的 Go 模块集合。
- **Durable success:** 数据和引用已经通过正式 owner 持久化，而不只是内存调用返回成功。
