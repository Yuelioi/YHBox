# 配置与资源持久化目标架构

> 状态：R1–R7 存储基座已实现并通过 Windows production migration smoke；installation/consent 等
> 后续领域仍按 Stage M 独立演进。

## 决策摘要

采用“一个平台 RootSet、两个事务一致性域、一个文件 CAS、多个明确生命周期”的混合架构：

1. **Content Catalog 使用 SQLite**：保存可查询的小型内容元数据、引用、版本、quarantine、migration
   状态和 GC mark；应用只暴露领域 Repository，不向 service 泄漏 SQL。
2. **Run Ledger 使用独立 SQLite**：Run summary、追加 journal/value index 属于高频、可按保留期归档的
   operational state，不能让它的 WAL、备份和清理拖累用户内容 Catalog。
3. **大字节保留文件 CAS**：Blob 仍以 `sha256:<hex>` 为稳定身份；数据库只保存 digest、size、media type、
   physical generation 和引用。不要把图片、录制、插件 payload 塞进 settings 或主 Catalog BLOB。
4. **Workflow Source 仍是正式可移植文档**：编辑/导入/导出契约保持 format/version；Catalog 提供列表、
   搜索、revision 和引用索引。是否把活动 Source bytes 存在 Catalog，留到实现基准后决定，但 bundle/export
   永远不能依赖本机 Catalog 内部 schema。
5. **Program 是 cache，不是 backup 内容**：按 compiler/catalog identity 可删可重建，独立配额与 LRU。
6. **Run 是追加型历史**：summary、event、value/blob reference 分表/分段，不再每个事件重写整个 JSON。
7. **settings 只放小型偏好**：安装 profile/consent 迁到 installation registry；secret 继续留 OS secure store。
8. **根级 manifest 与 migration engine 统一生命周期**：所有正式数据布局变化经过注册步骤、
   dry-run、备份、commit journal、resume/recovery；不再在任意 Store 构造器里偷偷重命名目录。

SQLite 不是“万能文件柜”。它负责关系、查询和事务；CAS 负责不可变大对象；portable artifact 负责交换；
cache、log、temp 负责可丢弃生命周期。

## 根目录

Windows 正式默认根建议为 `%LocalAppData%\Yotta\Yotta\`；当前全部设置也先留在本机根，只有未来明确
需要跨设备漫游的小型偏好才建立单独的 `%AppData%` adapter。开发/便携模式必须通过显式启动参数选择，
不能再从 exe 位置隐式判断。

```text
<root>/
  root.json                 # 小型、可人工诊断的 data-root identity/layout header
  config/
    settings.json           # generation/checksum envelope；仅小型偏好
  catalog/
    content.db              # workflow/asset/schedule/install metadata/reference authority
    migrations/             # migration journal 与失败报告，不放 SQL source of truth
  state/
    runs.db                 # append-only Run ledger；独立 retention/backup/checkpoint
  objects/
    sha256/
      ab/cd/<digest>         # immutable CAS，分片目录；物理布局不进入外部引用
  documents/
    exports/                 # 用户明确导出/可移植文件；不作为活动索引
  packages/
    node/                    # node package generations/trust authority
  cache/
    programs/                # 可重建 compiler artifacts
    previews/                # thumbnails/derived media
    indexes/                 # 可选可重建 projection；若 Catalog index 已足够可不建
  diagnostics/
    logs/
    crashes/
    captures/
  backups/
    migrations/
  tmp/
```

逻辑根由一个 `storage.Roots` 值对象提供：

```go
type Roots struct {
    Root, Config, Catalog, State, Objects, Packages, Cache, Diagnostics, Backups, Temp string
}
```

只有 composition root 可以创建 `Roots`。业务 service 不拼接 `<exe>/data`、`logs` 或环境变量。测试通过
显式临时 `Roots`，CLI 通过 `--data-root` 覆盖同一 resolver。

### Config 与 cache 的平台映射

- 正式 Windows 根放 LocalAppData，因为 workflow、asset、run、安装 profile 都是设备本地且可能很大。
  不能直接使用 Go `UserConfigDir` 作为总根，它在 Windows 指向可能被重定向的 Roaming AppData。
- 当前 settings 含大量设备相关项，R2 先整体进入 LocalAppData。未来确有跨设备漫游时，只把经过专门
  挑选的小型偏好投影到 Roaming/AppData；绝不默认漫游 CAS、Run、target identity 或日志。
- cache/diagnostics 在产品层标为不进入用户内容备份；卸载/清理可独立删除。
- 便携模式使用 `<explicit-root>`，必须写 `root.json` 且 UI 明示；不以“exe 旁发现 data”自动进入。

## 深模块与接口

不要创建一个泛型 `Store[T]`。真正稳定的 seam 是不同 durability/transaction adapter 会变化的地方：

```text
internal/storage/
  roots          解析/验证根路径与 single-writer lease
  catalog        打开 content/run DB、事务、schema migration、backup、integrity
  objectstore    immutable CAS、pin、physical generation、verify
  lifecycle      quota、retention、GC、quarantine、health
  migrate        data-root step registry、preflight、resume/recovery

domain repositories
  workflowrepo   Source metadata/revision/reference + portable artifact
  assetrepo      Global Asset metadata/variant/reference/query
  runrepo        Run summary + append-only event/value access
  installrepo    profiles/consents（secret 仅保存 secure-store key）
  schedulerepo   schedule transaction/query
```

Service 依赖领域动作而不是文件或 SQL：

```go
type AssetRepository interface {
    Query(context.Context, AssetQuery) (AssetPage, error)
    Get(context.Context, AssetID) (Asset, error)
    Commit(context.Context, ExpectedRevision, AssetMutation) (Asset, error)
}

type RunRepository interface {
    Create(context.Context, NewRun) (RunSummary, error)
    Append(context.Context, RunID, ExpectedGeneration, []RunEvent) (RunSummary, error)
    Query(context.Context, RunQuery) (RunPage, error)
    ReadEvents(context.Context, RunID, EventCursor, int) (EventPage, error)
}
```

Repository 隐藏 SQL schema、statement 和 object path。事务不能跨 Repository 被 UI 随意组合；需要跨域
原子性时由 application command 进入一个明确的 Unit of Work。

## Catalog 边界

首版 Content Catalog 建议包含：

- `meta(key, value)`：application id、schema/layout version、created/updated build，仅放机器可读根信息。
- `assets`、`asset_variants`、`asset_tags`：索引字段与 Blob refs。
- `workflow_sources`、`workflow_refs`、`workflow_quarantine`：列表/revision/reference/recovery。
- `schedules`、`snippets`：小型事务数据。
- `installations`、`consents`：本机 profile/authority；credential 只存 secure-store logical key。
- `object_refs`：`owner_kind + owner_id + role -> digest` 的权威引用边。
- `gc_objects`：observed size、state、unreachable_since、physical generation。
- `migration_history`：step、ordinal、checksum、source/target layout、started/committed、backup id、error report。
- `quarantine`：领域、原 identity、原因、隔离时间、可选原始 bytes digest。

Run Ledger 建议包含 `runs`、`run_events`、`run_values` 和自己的 migration/retention metadata。它只引用
不可变 workflow/program/object digest，不参与修改用户内容，因此无需伪造跨库原子事务：先确认所需不可变
事实存在，再创建 Run；中间崩溃至多留下未使用对象或未创建 Run。

禁止将以下内容放进 Content Catalog：大图片/视频/录制 payload、插件 generation 文件、日志、调试截图、
可随时重建的完整 Program bytes。小 JSON 不因为“可读”就继续一对象一文件；大 bytes 不因为“有数据库”
就全部变成 BLOB。

settings 不使用数据库：它是带 `schema_version`、单调 `generation`、payload checksum 的小型 envelope。
平台 `AtomicFileWriter` 负责同目录随机 staging、sync、发布与 backup；启动时从 target/backup/staging 中选择
可验证且 generation 最新的完整版本，损坏对象进入 recovery，不能静默把整份用户配置换成默认值。

## 事务与 Blob 提交协议

跨 Catalog/CAS 不存在真正的单文件事务，因此固定顺序：

1. 流式写 CAS staging，计算 digest、校验上限，flush/sync。
2. 原子发布不可变对象；重复 digest 验证 size/bytes 后复用。
3. 开 Catalog write transaction，以 expected revision/generation 校验并提交 metadata + `object_refs`。
4. 若第 3 步失败，Blob 成为暂时孤儿，由宽限期 GC 回收；绝不先提交引用再写 Blob。
5. 返回成功前完成定义好的 durability barrier；若“已发布但未确认 durability”，返回带 commit outcome 的错误，
   application 必须重新读取事实，不能盲目重试。

进程级 `Roots.Lease` 保证同一 data-root 只有一个 writer。SQLite 自身仍设置 busy timeout 和短事务；
不允许网络共享目录；后台查询不得持有长 read transaction 阻塞 checkpoint。

## SQLite 策略

- 两个 DB 使用不同 `application_id`；打开时校验 `application_id`、`user_version`、required tables/indexes；
  未知更高版本只读或拒绝，绝不降级写。
- 生产默认考虑 WAL + `synchronous=FULL`，但必须基准验证 Wails 单进程读写；如果并发收益不明显，
  rollback journal 更简单。WAL 是数据库持久状态的一部分，备份不能只复制 `catalog.db`。
- 每个 write command 使用短事务；Run event 批量 append，避免逐事件 fsync。
- migration 使用显式 SQL/Go step registry；已发布 step 的 ID/ordinal/checksum 不可修改。每步完成验证后在
  事务内更新 ledger 与 `user_version`；文件变换另由根级 journal 协调。
- 定期运行 `quick_check`，显式维护 checkpoint/backup；不要在每次启动做完整 `integrity_check`。
- 备份使用 SQLite Online Backup API 或 `VACUUM INTO` 生成一致快照，再把 `root.json`、package authority 和
  migration 需要的 CAS live set 写入 manifest。不能对活动 WAL 数据库普通复制单个主文件。

## GC、配额与清理

对象删除采用延迟 mark/sweep：

1. 在 Catalog 一致快照中枚举全部 `object_refs`、运行中的 pins、undo/recovery/backup roots。
2. 无引用对象写入 `unreachable_since`，本轮不删。
3. 越过宽限期后，在新的写事务/lease 下再次确认仍不可达。
4. 先从活动 physical generation 移到 trash/quarantine generation，再异步删除；记录 reclaimed bytes。

配额分三层：

- 根总量 soft/hard limit。
- content、run-history、cache、diagnostics 分区预算。
- 单对象/单记录/单操作预算。

达到 soft limit 时先清 cache/过期日志/过期 Run，再报告可预览的 content GC；达到 hard limit 时阻止新增大
对象，但仍允许删除、导出和修复。用户内容绝不与 cache 一起静默清除。

## Migration engine

根布局版本与产品版本分开，例如 `layoutVersion: 1`，不出现 `3.1` 目录名。注册步骤是有向无环的确定链：

```go
type Step interface {
    ID() string
    From() LayoutVersion
    To() LayoutVersion
    Preflight(context.Context, Snapshot) (Plan, error)
    Apply(context.Context, Plan, Journal) error
    Verify(context.Context, Plan) error
}
```

执行阶段：

1. 获取 root writer lease，读取 `root.json` 和 migration journal。
2. preflight：空间、权限、未知文件、schema、Blob refs、目标版本链；只读生成计划。
3. 创建一致 backup manifest；对不可重建 authority 做 snapshot，不复制 Program/cache/log。
4. 写 journal `prepared`，逐 step 执行可重复或可检测的动作。
5. 验证新 Catalog、引用完整性、对象数量/摘要；原子发布新 `root.json`。
6. journal `committed` 后才允许清理旧布局；失败时下次 resume 或进入 recovery UI。

开发期不为未发布的临时格式写 reader；正式 release 之后，每个存储格式变化必须同时提交 migration step、
fixture、dry-run golden、kill-point recovery test 和 version inventory 更新。

## 分阶段实现

### R2 — Root 与 lifecycle 基座

- 新增 `storage.Roots`、Windows LocalAppData resolver、显式 portable/dev override、root identity。
- GUI/CLI/settings/log/debug 统一从 Roots 获取路径；settings 加 version/generation/checksum recovery envelope。
- 根级 single-writer lease、health report 与版本 inventory。
- 暂不搬现有数据；测试使用新根，开发数据由后续 migration 处理。

### R3 — Catalog foundation

- 选择并锁定 Go SQLite driver，完成 Content Catalog/Run Ledger application id、user version、transaction、
  backup、quick-check、migration registry 与故障注入测试。
- 先迁低耦合的 Schedule/Snippet/installation metadata，验证 Repository seam。

### R4 — Asset catalog + CAS v2

- Asset metadata/query/reference 进入 Catalog；CAS 改为分片物理布局并接 object refs、宽限 GC、分区配额。
- Global Asset 与 Workflow Resource 语义继续分离，但共享 object bytes。
- 用 10k metadata、100k Blob 与崩溃矩阵验收。

### R5 — Workflow 与 cache

- Workflow Source 列表/revision/reference/quarantine 进入统一 Repository；portable source/bundle 契约不变。
- Program 迁入独立 cache lifecycle，按 build identity/LRU 回收。

### R6 — Run history

- 独立 Run Ledger 的 summary/event/value 追加模型；分页查询、retention/archive、payload CAS。
- 保留现有 Run domain 状态机与 journal 事实，只更换 durable representation。

### R7 — 正式 data migration 与恢复 UI

- 为首个公开版本冻结旧布局 fixture。
- 实现 dry-run、空间估算、自动 snapshot、resume/rollback、quarantine 管理、导出诊断。
- 真机从安装/升级/断电 kill-point 到新版本 smoke；旧布局清理需单独确认。

## 拒绝的方案

- **所有内容继续一对象一 JSON**：易调试，但启动扫描、跨对象事务、索引、迁移与 GC 会继续复制实现。
- **所有内容都放一个 SQLite BLOB**：事务简单，但大媒体、流式访问、CAS 去重、插件 generation、备份体积和
  故障域变差。
- **每个功能一个 SQLite**：看似隔离，实际把跨域引用、backup、migration、GC 和连接管理复杂度放大。
  只按一致性与生命周期分为 Content Catalog 和 Run Ledger，不按 Asset/Schedule/Snippet 继续拆库。
- **目录名绑定产品版本**：产品频繁升级不等于存储布局每次升级；会制造无意义搬家和兼容分支。
- **启动时发现旧目录就直接 rename**：没有 dry-run、backup、journal、resume 与空间/权限检查，不能作为
  上线后的 migration 机制。

## 一手依据

详细的事实/推论分离与 66 个官方链接见
[配置与资源持久化：一手资料研究](storage-config-primary-research.md)。关键依据包括 Microsoft Known
Folders、Go `UserConfigDir`/`UserCacheDir` 与 `os.Rename` 合同、SQLite WAL/Backup/PRAGMA 文档，以及
Git/OCI/containerd 的 CAS、引用和 GC 设计。
