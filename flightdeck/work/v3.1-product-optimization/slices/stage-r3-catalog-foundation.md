# R3 — Catalog foundation

## Outcome

在 RootSet 下建立两个独立、可验证、可升级和可一致备份的 SQLite 一致性域：

- `catalog/content.db`：Content Catalog，保存后续内容类 Repository 的小型事务数据和索引。
- `state/runs.db`：Run Ledger，保存后续高频追加、独立保留期的运行数据。

本切片只建立数据库生命周期和深模块边界，不迁移 Schedule、Snippet、Asset、Workflow 或 Run 领域数据。
当前开发数据无需兼容；正式发布后的升级由不可变 migration registry、备份和验证流程承载。

## Driver decision

锁定 `modernc.org/sqlite v1.54.0`：

- 使用 `database/sql`，不要求 CGO 或 Windows GCC，覆盖当前 Windows 完整支持及 Linux/macOS 预览构建。
- driver 提供 SQLite Online Backup API、连接级 PRAGMA 和 transaction lock 支持。
- v1 模块和 Go module checksum/pinning 比仍处于 v0 的备选实现更适合作为长期持久化 adapter。

未选择 `github.com/mattn/go-sqlite3`，因为它把 CGO/GCC 变成桌面构建和交叉编译前提；未选择
`github.com/ncruces/go-sqlite3`，因为它当前仍是 v0 且 wasm sandbox 会增加每连接内存成本。SQLite adapter
只存在于 `internal/storage/catalog` 内，service 和后续领域 Repository 不接触 driver 或 `*sql.DB`。

## Identity and schema

- Content Catalog `application_id = 0x594F5443`（`YOTC`），Run Ledger
  `application_id = 0x594F5452`（`YOTR`）；两者是 Yotta 暂用且未向 SQLite 官方登记的标识。
- 两库分别维护自己的 `user_version`、不可变 migration ID/ordinal/checksum 和 required schema。
- 新空库只在一个事务中领取 identity 并应用 schema；未知 application ID、未来 user version、
  migration checksum 漂移或缺少 required schema 时 fail closed，绝不覆写或降级。
- 生产连接使用短事务、foreign keys、busy timeout、WAL 与 `synchronous=FULL`。WAL 是数据库状态的一部分，
  备份必须走 Online Backup API，不能复制活动主文件。

## Module seam

`catalog.Foundation` 一次拥有两个数据库及连接池，向 composition root 暴露：

- `Open` / `Close`：完成 identity、migration、连接策略和生命周期。
- `Check`：返回 application/schema、journal、SQLite version、page/freelist、`quick_check` 和
  foreign-key 检查结果。
- `Backup`：把两个数据库在线备份到一个新 backup set，并在两个快照均验证后最后发布 manifest。

后续 R4/R6 在模块内提供领域 Repository；不增加通用 SQL accessor，也不允许 UI/application service
自行拼装跨 Repository transaction。Content Catalog 和 Run Ledger 是两个故障域，backup set 是两个有时间戳
的独立一致快照，不伪装成跨库原子快照。

## Verification

- [x] 新根创建两个 identity 不同的数据库，重复打开保持 schema 和 migration ledger 稳定。
- [x] 外来库、未来 schema、被篡改 migration checksum、缺表和损坏页全部 fail closed。
- [x] migration commit 前故障可安全重开；已提交 migration 不会重复执行。
- [x] `quick_check` 与 foreign-key 检查生成结构化报告。
- [x] WAL 活动库通过 Online Backup API 产生两个可独立打开、identity/schema 正确的快照；manifest 最后发布。
- [x] adapter 在 `CGO_ENABLED=0` 下可测试/构建。
- [x] GUI、CLI 与开发入口统一拥有并关闭 Foundation；版本清单显示两个 schema 版本。
- [x] 按 Git 变更范围运行 `task check`，由持续后台 wrapper 保留同一进程最终退出码。

## Result

2026-07-25 完成。`modernc.org/sqlite v1.54.0` 已成为直接依赖，许可证检查通过。双库使用 WAL/
`synchronous=FULL`，启动严格验证 identity、schema、required objects 与完整 migration ledger；只读 CLI
health 同时报告两库。Online Backup 测试在活动 WAL 存在时验证 snapshot，并以流式 SHA-256 和 manifest-last
发布，注入 manifest 前故障不会留下可误认的备份集。

定向 Go test/vet、`CGO_ENABLED=0`、race 均通过；持续后台 `task check` 保留同一进程退出码并返回 0。
`task build` 返回 0，Windows GUI metadata 与隔离 5 秒启动 smoke 通过。production CLI 在新的隔离 profile
创建 `content.db`/`runs.db`，随后只读 health 返回两个数据库 `healthy=true` 且默认隐藏物理路径。

## Primary references

- [modernc SQLite package](https://pkg.go.dev/modernc.org/sqlite)
- [ncruces SQLite driver](https://pkg.go.dev/github.com/ncruces/go-sqlite3/driver)
- [mattn SQLite driver](https://github.com/mattn/go-sqlite3)
- [SQLite application_id and user_version](https://www.sqlite.org/pragma.html#pragma_application_id)
- [SQLite Online Backup API](https://www.sqlite.org/backup.html)
- [SQLite quick_check](https://www.sqlite.org/pragma.html#pragma_quick_check)
- [SQLite registered application IDs](https://sqlite.org/src/artifact?ci=trunk&filename=magic.txt)
