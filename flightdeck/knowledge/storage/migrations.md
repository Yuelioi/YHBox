# Storage and migration changes

## Find the authority first

不要从现有文档或磁盘样本猜路径。正式 profile 投影来自 `internal/storage.Resolve`；当前主要 owner 是：

| Data | Owner |
| --- | --- |
| root identity、目录投影、single-writer lease | `internal/storage` |
| Content Catalog / Run Ledger schema 与 backup | `internal/storage/catalog` |
| Workflow Source / Program cache | `internal/workflowstore` |
| Blob bytes / layout / GC lease | `internal/blob` + Catalog object repository |
| settings generation/recovery | `internal/services/settings_store.go` |
| schedule/snippet 小型 store | 对应 `internal/services` package |
| Node Package generations/registry | `internal/nodepackage` |
| root layout migration/recovery | `internal/storage/migrate` |

版本值从所属 package 读取，聚合检查使用 `task versions:inventory`。公开 writer 版本还会被冻结在
`contracts/releases/<product>/version-domains.json`；`task versions:compatibility:check` 要求当前代码继续声明
可读所有仍受支持的已发布版本。不要在无关 store 或文档中增加第二个 schema 常量。

## Changing a durable format

1. 先确定这是 public durable authority、可删除 cache，还是未发布开发 fixture。4.0 之前未发布格式不自动
   获得兼容承诺；从 4.0 floor 起的 writer 必须有受支持的升级路径。
2. shape/语义变化提升所属 identity/version，不在旧 identity 下原地解释新字段。
3. 为公开格式添加确定性的相邻 migration。输入先 strict inspect/preflight，转换写到 staging/copy，验证当前
   schema、引用、digest 和 identity 后再 durable publish。
4. root/Catalog 跨 authority 迁移使用 registry、backup set、journal 和 resume/rollback；不要让 service 在普通
   open 时各自偷偷迁一部分。
5. 兼容 reader 成功读取旧格式后必须 durable rewrite；只在内存改版本会让每次启动重复迁移，也没有 reader
   退役点。
6. 未知/未来版本、checksum drift、identity mismatch 和损坏保留原事实并 fail closed 或 quarantine；不能用
   默认值覆盖。
7. 提升公开 writer 前，把旧版本加入 owner 的真实 reader/migration，并在 inventory 的 `ReadableVersions`
   声明它；声明只用于发现退化，不能替代旧 bytes、durable rewrite 和关闭重开测试。

SQLite 开启 WAL 时不能靠复制活动 `.db` 文件备份。使用 Catalog 的 Online Backup 边界，并在所有数据库
快照验证后最后发布 backup manifest。Blob bytes 的清理必须结合 Catalog references 和 active leases。

## Required tests

- 冻结旧 bytes/数据库 fixture；测试 preflight、单步和链式 migration。
- 在 publish 前、rename/manifest 边界和 journal 阶段注入 kill/failure，验证 resume 或 rollback。
- 完成后关闭并重新打开真实 owner，不能只断言迁移函数返回的内存对象。
- 测未来版本、错 identity、unknown field、损坏 checksum、引用缺失和 quota。
- root/Catalog/Run/Blob migration 运行 `task smoke:storage-migration`；并发或 writer lease 变化运行 race test。
- 最后运行 `task check`。发布候选还会通过 `task package` 执行 full gate 和 frozen payload smoke。
- 首次发布新产品版本时，完成上述测试后执行 `task versions:compatibility:freeze`；历史 snapshot 是发布事实，
  不能为了让门禁通过而修改。

手工诊断使用 CLI `health`/`migrate` 命令；命令和 single-writer 注意事项见
[CLI reference](../../../docs/reference/cli.md)。
