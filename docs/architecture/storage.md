# Storage consistency

## RootSet 与进程所有权

所有正式持久化路径由 `internal/storage` 一次解析并由 composition root 下发。Windows 正式默认根是
`%LocalAppData%\Yotta\Yotta`；显式参数优先于 `YOTTA_ROOT`，两者都没有时才使用平台默认。不得通过
exe 目录、当前工作目录或“发现旧 data”推断模式。`task dev` 使用仓库 `.task/yotta-dev` 隔离 profile，
CLI 的 `--data-root` 覆盖整个 profile，不能单独拆出 settings 根。

`root.json` 固定 `yotta.storage-root/<layoutVersion>`，产品版本与存储布局版本独立。进程打开 profile 后持有
根级 writer lease，GUI/CLI 不能同时写同一根。未知 identity、未知布局和非空未认领目录都 fail closed；
打开失败不修改、不迁移、不删除目录内容。

当前 layout 1 的主要生命周期目录是：

- `config/`：小型设置；
- `data/`：当前文件型领域 Store；
- `objects/sha256/`：不可变内容对象；
- `catalog/`、`documents/exports/`：Content Catalog 与用户明确导出的可移植文档；
- `packages/`、`cache/`、`state/`：安装包、可重建缓存、运行状态；
- `diagnostics/`：日志与调试截图；
- `backups/migrations/`：未来布局迁移的一致备份；
- `runtime/`、`tmp/`：writer lease 与临时内容。

`yotta health` 只读统计各生命周期的对象数和 bytes，并报告 staging/recovery/未知顶层项；同时验证
Content Catalog 和 Run Ledger identity、schema、journal、`quick_check` 与 foreign keys。默认隐藏物理路径，
显式 `--show-path` 才输出。

## Content Catalog 与 Run Ledger

`internal/storage/catalog` 独占 SQLite adapter 和连接池，不向 service 暴露 `*sql.DB`。后续领域
Repository 在该模块内增加窄接口；UI/application service 不拼 SQL，也不自行组合跨库 transaction。

- `catalog/content.db` 使用 `application_id=0x594F5443`（`YOTC`），属于内容元数据的一致性域。
- `state/runs.db` 使用 `application_id=0x594F5452`（`YOTR`），属于高频追加和独立保留期的一致性域。
- 两库分别维护 `user_version`、required schema 和不可变 migration ledger；未知 identity、未来 schema、
  migration checksum 漂移或缺少 required object 时 fail closed。
- 连接使用 foreign keys、busy timeout、WAL 与 `synchronous=FULL`。短 write transaction 由后续
  Repository 拥有，不能把跨库写伪装成原子事务。
- `Foundation.Check` 使用 `quick_check` 和 `foreign_key_check`；完整 `integrity_check` 只用于明确诊断/
  恢复，不放进每次启动。
- `Foundation.Backup` 使用 SQLite Online Backup API 分别生成两个一致快照，流式计算 SHA-256，并在两个
  snapshot 都通过 identity/schema/quick-check 后最后发布 backup-set manifest。活动 WAL 数据库不能通过
  复制单个 `.db` 文件备份。

当前两个 application ID 是未向 SQLite 官方登记的 Yotta 暂用值；公开发布前需再次核对官方登记表并决定
是否申请登记。当前 schema 都是 1，只包含 `meta` 与 migration ledger，不迁移开发期旧领域数据。

## Settings

Settings 使用 immutable snapshot：更新流程是 clone、mutate/merge、validate、atomic swap、atomic save。
磁盘格式是 `yotta.settings/1` envelope，包含单调 generation、canonical payload 与 domain-separated SHA-256
checksum。同目录随机 staging 写入并 sync，发布前保留最近有效 `.bak`，原子替换后同步父目录。启动从
primary、backup、staging 中选择 generation 最新且 checksum 有效的唯一值；全部损坏或同代冲突时返回
recovery-required，不以默认设置覆盖事实。后续成功保存会先把损坏 primary 保留到 `config/recovery/`。

Workflow Source、Program 与 Run 是分离且独立版本化的 durable artifact。各 Store 只接受当前所属 contract，以 canonical bytes、内容摘要和 revision/generation CAS 约束更新；写入采用同目录临时文件、sync、原子替换和父目录 sync。内存状态只在 durable publish 后更新；rename 已提交但目录 durability 未确认时返回显式 committed-warning，不生成第二个 identity 或伪装失败。

Blob Store 独占 immutable content-addressed bytes，发布前验证 digest/size/quota，引用与租约决定 GC 可达性。Stream 和 Resource lease 只属于 Run 生命周期，不能进入 Workflow Source、Program 或 durable Run value。Run Store 持久化 QUEUED/RUNNING/terminal 状态、NodeAttempt 与 AdapterAction；重启只把遗留 RUNNING 转为 INTERRUPTED，不透明重放副作用。

Node Package Store 使用另一套严格的 registry-last generation 模型：验证后的不可变 generation 先在同一文件系统 durable publish，canonical `registry.json` 最后原子替换并成为唯一安装 authority。registry v2 同时持有 monotonic trust policy、publisher signature evidence、namespace ownership、revocation/quarantine 与 package pointers，避免 trust 和 generation authority 跨文件提交。registry 写入 rename 前失败不发布内存；rename 已提交但目录 durability 未确认时按 `durablefs.Committed` 发布同一代并返回 warning。incoming/orphan generation 没有 authority，reopen 时清理；所有 registry 引用的 generation 都重新验证 manifest、精确文件集合、size、SHA-256、mode、Ed25519 signature 和当前 trust status。

## 发布版升级协议

开发期旧 `bin/data`、`bin/settings.json` 不属于 layout 1，不自动兼容，也绝不由启动过程删除。首个正式发布版
冻结后，任何持久化布局变化必须在同一变更中完成：

1. 提升独立 layout/schema version，并更新 `yotta-versions inventory`；
2. 注册不可变 migration step：稳定 ID、唯一 `from -> to`、受审实现/fixture checksum；
3. 提供旧版 fixture、只读 preflight/dry-run golden、空间与权限检查；
4. 在 writer lease 下创建一致 backup manifest，写 prepared journal 后才执行；
5. Apply 必须可重入或能检测已完成状态；Verify 校验 schema、计数、摘要和引用完整性；
6. 原子发布新 `root.json`，journal committed 后才允许单独清理旧布局；
7. 覆盖 staging、publish、verify 等 kill-point 的 resume/recovery 测试。

`storage.MigrationRegistry` 现在已经锁定“唯一、连续、只向前”的根布局路由规则：分叉、缺步、倒退、
无效 checksum 和高于当前应用的根都会被拒绝。Catalog schema migration 与一致 backup foundation 已落地；
根布局的数据迁移执行器及 recovery UI 在 R7 实现，Store 构造器不得自行 rename 或暗中兼容旧格式。
