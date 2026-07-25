# 配置与资源持久化：一手资料研究

日期：2026-07-25
范围：Yotta R1 配置与资源持久化审计的外部依据；本文不代表已完成仓库现状审计，也不直接决定迁移实现。

## 研究口径

- 只采用平台/项目自身的一手资料：Microsoft Learn、Go 官方文档与源码、SQLite 官方文档、XDG
  规范、Git 官方文档、OCI 规范及 containerd 项目设计文档。
- 每节先列“来源事实”，再列“对 Yotta 的推论”。推论是架构建议，不冒充上游保证。
- 本文讨论的是持久化边界和故障模型，不把目录命名美化当作存储架构。

## 结论摘要

对 Yotta 最合适的方向不是“全部继续写 JSON”，也不是“全部塞进一个 SQLite”：

1. 建立平台 `DataRoots`，把配置、权威数据、状态、缓存、日志、运行时临时物分开；Windows
   的大量本机数据默认落在 `%LOCALAPPDATA%`，只有明确需要漫游且足够小的偏好才进入
   `%APPDATA%`。
2. 用 SQLite 承担可查询、需事务一致性的 metadata、稳定资源身份、引用关系、任务/运行索引和
   migration ledger；大 Blob 不进入 SQLite 通用表。
3. Blob 使用不可变 CAS；descriptor 至少包含算法、摘要、字节数和媒体类型。资源的用户身份
   `resource_id` 与内容身份 `digest` 分离，多个资源或版本可以引用同一内容。
4. SQLite 和 CAS 之间采用“先落盘并校验不可变对象，再在数据库事务中发布引用”的提交次序；
   崩溃产生的无引用对象由带租约和宽限期的 mark-and-sweep GC 回收。
5. 小配置文件通过平台化 `AtomicFileWriter` 写入。Go 官方明确说 Windows 上 `os.Rename`
   不保证原子，因此不能把“临时文件 + `os.Rename`”宣称为跨平台原子写。
6. 存储 schema 版本独立于产品版本。`PRAGMA user_version` 只作快速兼容门，详细历史由内置、
   有 checksum 的 migration registry 记录；迁移流程必须具备 preflight、dry-run、备份、
   apply、verify、commit 和 recovery。

## 1. 平台数据根与配置分层

### 来源事实

- Windows 的 `FOLDERID_LocalAppData` 和 `FOLDERID_RoamingAppData` 都是 per-user Known
  Folder，默认分别为 `%LOCALAPPDATA%` 和 `%APPDATA%`。Microsoft 对新代码建议通过
  `SHGetKnownFolderPath` 和 Known Folder ID 获取位置，而不是使用已弃用的 CSIDL API。Known
  Folder 还可能被重定向，不能把示例物理路径当作恒定路径。
  [KNOWNFOLDERID](https://learn.microsoft.com/en-us/windows/win32/shell/knownfolderid)；
  [Known Folders](https://learn.microsoft.com/en-us/windows/win32/shell/known-folders)
- Windows 管理员可以把 `AppData/Roaming` 重定向到本机其他位置或网络共享。因此目录名
  “Roaming” 不是“永远是快速本地磁盘”的保证。
  [Configure Folder Redirection](https://learn.microsoft.com/en-us/windows-server/storage/folder-redirection/folder-redirection-using-group-policy)
- Go 的 `os.UserConfigDir()` 在 Windows 返回 `%AppData%`，在 macOS 返回
  `~/Library/Application Support`，在 Unix 使用 `$XDG_CONFIG_HOME`；`os.UserCacheDir()`
  在 Windows 返回 `%LocalAppData%`，在 macOS 返回 `~/Library/Caches`，在 Unix 使用
  `$XDG_CACHE_HOME`。两个 API 都要求应用创建自己的子目录。
  [Go `os` package](https://pkg.go.dev/os#UserConfigDir)；
  [Go `os` 源码](https://go.dev/src/os/file.go)
- XDG Base Directory Specification 明确区分 user data、configuration、persistent state、
  non-essential cache 和 runtime files；其中 state 可包含历史、最近使用、窗口布局和重启可恢复
  状态，而 cache 是非必要、可删除的数据。
  [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/latest/)
- Windows DPAPI 的 `CryptProtectData` 通常把密文绑定到同一登录用户和同一机器，并附带完整性
  校验；设置 `CRYPTPROTECT_LOCAL_MACHINE` 会改为本机范围，意味着同机其他用户也可解密。
  [CryptProtectData](https://learn.microsoft.com/en-us/windows/win32/api/dpapi/nf-dpapi-cryptprotectdata)

### 对 Yotta 的推论

Yotta 应先定义逻辑根，再由平台 adapter 映射物理路径；业务 service 不应自行拼 `%AppData%`、
`bin/data` 或工作目录。建议的逻辑分类如下：

| 逻辑根 | 内容 | Windows 默认建议 | 恢复/清理语义 |
| --- | --- | --- | --- |
| `config` | 小型用户偏好、功能开关、显式路径选择 | `%APPDATA%\Yotta\config`，仅对明确可漫游内容使用 | 保留、备份、schema 校验 |
| `data` | 工作流、资源 metadata、CAS、计划等权威本机数据 | `%LOCALAPPDATA%\Yotta\data` | 不可自动丢弃 |
| `state` | 会话、运行队列、最近项、恢复点 | `%LOCALAPPDATA%\Yotta\state` | 可按生命周期清理，但不能伪装成 cache |
| `cache` | 缩略图、派生索引、下载缓存、编译缓存 | `%LOCALAPPDATA%\Yotta\cache` | 必须可重建、可驱逐 |
| `logs` | 诊断日志 | `%LOCALAPPDATA%\Yotta\logs` | 按大小和天数轮转 |
| `runtime` | 锁、socket、in-progress staging | 本机临时/专用 runtime 根 | 启动恢复后可清理 |
| `secrets` | token、credential、授权凭据 | 平台凭据设施或 DPAPI 加密载荷 | 不进入可移植 workflow/CAS |

几个重要限制：

- 不能直接用 `os.UserConfigDir()` 作为 Yotta 所有数据的根；在 Windows 它指向可能被重定向的
  Roaming AppData，大型 CAS、SQLite WAL、运行记录和缓存应明确使用本机根。
- “便携模式”可以显式把这些逻辑根映射到用户选择的目录，但它是单独的部署模式，必须展示其
  备份、权限、网络盘和性能风险；不能通过当前工作目录静默触发。
- 安装包内置资源属于只读 application bundle；首次使用时需要修改的资源应复制/导入到 data
  store，而不是写回 exe 相邻目录。
- secret 只在 metadata 中保存不敏感的稳定引用；密文本身由平台 secret adapter 管理。DPAPI
  是 Windows adapter 的候选，不是跨平台文件格式。

## 2. 小文件原子写、并发和崩溃恢复

### 来源事实

- Go 的 `File.Sync()` 把文件当前内容提交到 stable storage；但 Go 对 `os.Rename` 明确说明：
  在非 Unix 平台，即使源和目标位于同一目录，也不是原子操作。
  [Go `File.Sync` 与 `Rename`](https://pkg.go.dev/os#Rename)
- 当前 Go Windows 实现最终调用 Windows rename 路径，但 Go 公共合同仍明确不提供 Windows
  原子保证；实现细节不能覆盖公共 API 的限制。
  [Go `file_windows.go`](https://go.dev/src/os/file_windows.go)；
  [Go Windows syscall source](https://go.dev/src/internal/syscall/windows/syscall_windows.go)
- Win32 `ReplaceFileW` 把一个文件替换为另一个文件，并可同时生成旧文件备份，且要求 replaced、
  replacement 和 backup 位于同一卷。它的 `REPLACEFILE_WRITE_THROUGH` 标志官方注明为“不支持”，
  并且若某些步骤失败，文档列出了多种可能的残留状态。
  [ReplaceFileW](https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-replacefilew)
- `MoveFileExW` 的 `MOVEFILE_WRITE_THROUGH` 会等待 move 实际落盘；跨卷 move 若允许，会退化为
  copy + delete。`FlushFileBuffers` 会把指定文件的缓冲数据写入设备，但频繁调用可能很昂贵。
  [MoveFileExW](https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-movefileexw)；
  [FlushFileBuffers](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-flushfilebuffers)

### 对 Yotta 的推论

需要一个深模块 `AtomicFileWriter`，封装而不是向调用方暴露临时文件、rename 和备份细节：

1. 在目标同目录创建随机 staging 文件，防止跨卷 copy + delete。
2. 完整序列化、校验大小/可解析性，`Sync` 后关闭 staging。
3. 平台 adapter 发布：Windows 对已有目标评估 `ReplaceFileW`，首次创建评估
   `MoveFileExW`；Unix adapter 使用本平台的 rename/fsync 语义。具体保证必须由故障注入测试
   证明，不由共同接口夸大。
4. 配置 envelope 保存 `schema_version`、单调 `generation` 和 payload checksum。启动恢复时可
   在 target、backup、staging 中选择“可解析、checksum 正确、generation 最新”的完整版本。
5. 单个配置只允许一个进程级 writer；并发 UI 保存通过 service 内串行化或 compare-and-swap
   generation，避免最后写入静默覆盖。
6. 不原地 truncate 后写 JSON；不把“调用返回成功”与“断电后绝对不会丢最后一次写入”等同。

这一方案的重点是可恢复状态机。即使底层 API 在异常阶段留下多个文件，启动恢复也能机械判定，而
不是要求用户从半截 JSON 中修复。

## 3. SQLite 作为 metadata、引用和索引层

### 来源事实：事务与 WAL

- SQLite 的 atomic commit 目标是让一个事务的全部修改同时发生或全部不发生；官方说明在 OS
  crash 或掉电中断时事务仍表现为原子，但这一保证依赖文件系统锁、flush 和硬件按合同工作。
  [Atomic Commit in SQLite](https://www.sqlite.org/atomiccommit.html)
- WAL 允许 reader 与 writer 同时工作，但只有一个 writer；wal-index 依赖共享内存，所以所有
  reader 必须位于同一台机器，WAL 不适合网络文件系统。WAL 还引入 `-wal`、`-shm` 和 checkpoint。
  [Write-Ahead Logging](https://www.sqlite.org/wal.html)
- 默认自动 checkpoint 阈值为 1000 页；持续存在的长 read transaction 可能让 checkpoint 无法完成，
  使 WAL 持续增长。WAL 中 `synchronous=FULL` 会在每次 commit 同步 WAL；`NORMAL` 省略这个
  commit-time sync，掉电后最近事务可能回滚，但 checkpoint 仍需同步。
  [WAL checkpoint and performance](https://www.sqlite.org/wal.html#performance_considerations)
- WAL 模式下，每个 `ATTACH` 数据库的事务各自原子，但跨多个 attached database 的整体事务不保证
  同时原子。
  [WAL overview](https://www.sqlite.org/wal.html#overview)
- `-wal` 是数据库持久状态的一部分。数据库与 WAL 分离可能丢失已提交事务或造成损坏；不能在活跃
  事务期间只复制主 `.db` 文件。
  [Write-Ahead Logging — WAL file](https://www.sqlite.org/wal.html#the_wal_file)；
  [How To Corrupt An SQLite Database](https://www.sqlite.org/howtocorrupt.html)
- `BEGIN IMMEDIATE` 立即启动写事务，若另一写事务已存在则返回 `SQLITE_BUSY`；显式 busy timeout
  可以让 SQLite 在锁冲突时等待到指定累计时长后再返回 `SQLITE_BUSY`。
  [SQLite transactions](https://www.sqlite.org/lang_transaction.html)；
  [sqlite3_busy_timeout](https://www.sqlite.org/c3ref/busy_timeout.html)

### 来源事实：备份、检查和空间

- Online Backup API 可增量复制 live database，只在实际读取的短时间持有 source lock；完成后
  destination 是复制开始时 source 的 snapshot。直接文件复制若遇到断电或活跃事务可能生成损坏
  备份。
  [SQLite Backup API](https://www.sqlite.org/backup.html)
- `VACUUM INTO` 生成最小化的一致 snapshot；与 Backup API 相比，它清除已删除内容并减少输出，
  但消耗更多 CPU、不能增量执行，且中断时输出文件可能不完整。
  [VACUUM](https://www.sqlite.org/lang_vacuum.html)
- 普通 `VACUUM` 通过临时数据库重建并覆盖原库，最多需要约原数据库两倍的额外可用空间；无显式
  `INTEGER PRIMARY KEY` 的表还可能改变 ROWID。
  [How VACUUM works](https://www.sqlite.org/lang_vacuum.html#how_vacuum_works)
- 默认 `auto_vacuum=NONE` 时，删除页进入 freelist 供后续复用，数据库文件不会自动缩小。
  `FULL` 会在每次提交移动/截断空闲页，可能增加碎片；切换 auto-vacuum 模式通常需要 `VACUUM`。
  [SQLite PRAGMA `auto_vacuum`](https://www.sqlite.org/pragma.html#pragma_auto_vacuum)
- `quick_check` 是 O(N) 的较快底层检查；`integrity_check` 还检查 UNIQUE 和索引一致性但为
  O(N log N)。二者都不检查外键，外键需 `foreign_key_check`。
  [SQLite integrity/quick/foreign-key checks](https://www.sqlite.org/pragma.html#pragma_integrity_check)

### 来源事实：连接安全和 schema

- SQLite 外键约束出于兼容原因默认可能关闭，应用应在每个连接明确设置；`trusted_schema`
  官方建议大多数应用在每个连接打开时设为 OFF。
  [SQLite Foreign Key Support](https://www.sqlite.org/foreignkeys.html)；
  [PRAGMA trusted_schema](https://www.sqlite.org/pragma.html#pragma_trusted_schema)
- `PRAGMA user_version` 是数据库 header 中留给应用的整数，SQLite 自己不使用；
  `application_id` 可标记数据库属于哪个应用格式。
  [SQLite PRAGMAs](https://www.sqlite.org/pragma.html#pragma_user_version)
- `schema_version` 是 SQLite 自己在 schema 变化时维护的内部值；官方警告错误修改可能让旧 prepared
  statement 在过时 schema 下运行，因此它不能代替应用 migration version。
  [PRAGMA schema_version](https://www.sqlite.org/pragma.html#pragma_schema_version)
- SQLite 只直接支持有限的 `ALTER TABLE` 操作；更复杂变化应使用官方给出的事务化重建流程，并在
  之后执行外键/完整性检查。
  [SQLite ALTER TABLE](https://www.sqlite.org/lang_altertable.html)

### 对 Yotta 的推论

SQLite 适合保存：

- Global Asset、Workflow Resource 的稳定 ID、名称、variant、标签、引用和 CAS descriptor；
- workflow 列表/搜索 metadata、计划、录制 metadata、插件/节点包安装状态；
- 运行记录和诊断索引，但高频、大体积事件 payload 应分层，避免无限膨胀主 catalog；
- schema migration、GC lease、import/export/migration job 的状态。

SQLite 不应保存：

- GB 级模板、录屏、宏原始流或任意 Blob 的通用字节；
- 可快速重建的缩略图字节（只保存 cache key/status 即可）；
- token 或未加密 credential；
- 为了“统一”而加入的整个 workflow 大 JSON BLOB，若查询/引用仍需每次全量反序列化。

运行策略建议：

- 权威 catalog 只放本机文件系统；不允许直接在 SMB/NFS/同步盘上以 WAL 多进程读写。
- 由一个 storage service 拥有连接池和写事务。写事务短小，显式设置 busy timeout；预知要写的
  流程用 `BEGIN IMMEDIATE`，让争用在事务开始时暴露。
- 每个连接机械设置 `foreign_keys=ON`、`trusted_schema=OFF`，并验证实际生效；WAL 和
  `synchronous=FULL`/`NORMAL` 的选择由崩溃测试与性能数据决定。不能为跑分快而设
  `synchronous=OFF`。
- checkpoint 是运维的一部分：记录 WAL bytes、checkpoint 结果和长 reader；在空闲/退出路径尝试
  有界 checkpoint，但不假设退出总会执行。
- 活库备份只走 Online Backup API 或 `VACUUM INTO`。备份完成后再写 manifest/checksum，并用
  原子发布协议进入 backup set；不使用 Explorer copy 或 `os.ReadFile` 复制活跃 `.db`。
- catalog snapshot 不等于完整资源备份。创建 backup lease 时应暂停 CAS sweep，完成 SQLite
  snapshot 后从该 snapshot 枚举全部 descriptor、复制或 pin 对应不可变对象并验证 digest，最后
  发布 backup manifest 后再解除 lease。
- 启动可运行低频 `quick_check`；升级、恢复和用户触发的深度诊断运行 `integrity_check` 加
  `foreign_key_check`。检查失败进入只读恢复模式，不继续写。
- 是否启用 `auto_vacuum=INCREMENTAL` 必须在建库前决定并基准测试。无论选择哪种，都要有
  freelist、DB/WAL 大小和维护预算；不能在用户交互路径无界 `VACUUM`。

## 4. CAS：对象、metadata、引用和索引分层

### 来源事实

- Git 把核心对象库描述为 content-addressable key-value store：写入内容得到唯一 key，loose
  object 以 hash 前两位为目录、其余为文件名。Git 另有 pack 与 index，把大量 loose object
  合并以降低磁盘、目录和备份负担。
  [Git Internals — Git Objects](https://git-scm.com/book/en/v2/Git-Internals-Git-Objects.html)；
  [git-prune-packed](https://git-scm.com/docs/git-prune-packed)
- OCI Content Descriptor 把 `mediaType`、`digest` 和 raw byte `size` 作为必需属性；从不可信来源
  取得内容后应验证大小和摘要。digest 使用 `algorithm:encoded` 形式，OCI 要求实现 SHA-256 验证。
  [OCI Content Descriptors](https://github.com/opencontainers/image-spec/blob/main/descriptor.md)
- containerd 将不可变 content store 与可变、可使用的 snapshot 分开；它用 GC reference labels
  表达 manifest/index 到子对象或 snapshot 的可达关系，从 root 保护整棵对象图。
  [containerd Content Flow](https://github.com/containerd/containerd/blob/main/docs/content-flow.md)
- Git `fsck` 分别报告 missing、unreachable、dangling 和 hash mismatch；Git GC 以可达性决定保留，
  对新对象使用时间宽限来缓解与并发 writer 的竞态，并明确说“立即 prune”会增加损坏风险。
  [git-fsck](https://git-scm.com/docs/git-fsck)；
  [git-gc](https://git-scm.com/docs/git-gc)

### 对 Yotta 的推论

#### 对象合同

统一 descriptor：

```text
BlobDescriptor {
  algorithm: "sha256"
  digest:    "sha256:<64 lowercase hex>"
  size:      int64
  mediaType: string
}
```

- `digest` 是不可变内容身份，不是用户素材身份；`resource_id`/`variant_id` 是可重命名、可授权、
  可追踪历史的领域身份。
- 目录可采用 `objects/sha256/ab/<remaining>` 或二级分片；具体分片深度由 100k/1M 对象 fixture
  在 NTFS、备份软件和杀毒开启下实测决定。
- Put 流程必须流式 hash 到 staging，核对 caller 提供的 size/digest，Sync 后用“不覆盖不同内容”
  的提交语义发布；若目标已存在则校验并复用。不要把整个 GB 对象读入内存。
- 图片、视频、压缩包等已压缩大对象默认不再压缩。大量极小 loose object 是否进入 pack 是后续
  优化，不应让首版 CAS 同时承担复杂 pack GC。

#### metadata 与查询

- SQLite catalog 保存 resource、variant、descriptor、引用边和 searchable metadata；CAS 文件只
  保存不可变 bytes。文件 sidecar 不是唯一 metadata 事实。
- UI 搜索、分页、hash 定位和引用检查走索引查询，不遍历对象目录；目录扫描只属于 repair/fsck。
- 可重建 thumbnail、OCR、visual feature 等派生物使用 `source_digest + transform_version` 做 cache
  key，版本变化时自然失效。

#### 跨 SQLite/CAS 提交

SQLite 事务不能原子覆盖普通 CAS 文件，因此采用可恢复顺序：

1. 建立有 TTL 的 ingest/migration lease。
2. staging 写完、Sync、核验 digest/size。
3. 发布不可变 CAS object；重复对象幂等复用。
4. SQLite 短事务写 descriptor、资源 metadata 和引用边。
5. commit 后释放 lease。

在第 3、4 步间崩溃只会产生无引用 CAS object，稍后可回收；反过来先提交 DB 再写对象会产生
“metadata 已发布但字节缺失”的用户可见损坏，风险更高。

#### GC、配额和修复

- GC 以所有权威 root 做 mark-and-sweep：workflow source/resource、global asset、run retention、
  schedule、安装包、显式 pin、backup/migration/import lease。
- 不把单独的 `ref_count` 当最终真相；它可作缓存，但崩溃后必须能从引用边重建。
- sweep 只处理“不可达且早于宽限期且无 lease”的对象；第一阶段移入 quarantine/记录 tombstone，
  下一维护周期才物理删除。GC 必须支持 dry-run、bytes/object 统计、取消和互斥。
- fsck 分层：快速检查 DB 引用的对象存在且 size 正确；深度检查重算 digest；repair 扫描孤儿与
  missing，绝不静默删除。
- 配额按类别统计 physical bytes 和 logical referenced bytes。达到软阈值先驱逐 cache、过期 staging
  和已过保留期 run artifact；权威 workflow/asset 不自动删除。Windows 可通过
  `GetDiskFreeSpaceExW` 获得当前用户实际可用空间（会考虑 per-user quota），写大对象前应保留安全
  水位。
  [GetDiskFreeSpaceExW](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-getdiskfreespaceexw)

## 5. Schema version 与 migration registry

### 来源事实

- SQLite 提供 `user_version` 和 `application_id` 字段，但不会替应用执行或记录迁移。
- 复杂 schema 变化需要显式事务、表重建和完成后的完整性检查。
- Live database 的一致备份应走 Backup API 或 `VACUUM INTO`，不能在可能存在 journal/WAL 的
  状态下只复制主文件。

来源：
[SQLite PRAGMA](https://www.sqlite.org/pragma.html#pragma_user_version)；
[SQLite ALTER TABLE](https://www.sqlite.org/lang_altertable.html)；
[SQLite Backup API](https://www.sqlite.org/backup.html)。

### 对 Yotta 的推论

#### 版本模型

- `storage_schema_version` 是单调整数，与 `3.1.x` 产品版本彻底解耦。
- `application_id` 固定标识 Yotta catalog；`user_version` 保存“已完整提交的最高 storage schema”
  作为快速开门检查。application ID 应先核对 SQLite 官方登记列表，避免随意撞号；打开未知 ID
  文件时拒绝迁移。
  [SQLite application ID registry](https://www.sqlite.org/src/artifact?ci=trunk&filename=magic.txt)
- 不读写 SQLite 内部 `schema_version` 作为应用迁移状态。
- 详细 ledger 至少包含：

```text
schema_migrations(
  migration_id TEXT PRIMARY KEY,
  ordinal INTEGER UNIQUE,
  checksum TEXT NOT NULL,
  state TEXT NOT NULL,
  app_build TEXT NOT NULL,
  started_at INTEGER NOT NULL,
  completed_at INTEGER,
  details_json TEXT
)
```

- migration 作为有序、不可变代码嵌入应用；已发布 migration 的 ID、ordinal 和 checksum 不可修改。
  修正只能追加新 migration。

#### 生命周期

1. **Discover**：读取 application ID、`user_version`、registry 和上次 migration job。
2. **Preflight/dry-run**：验证支持的源版本、完整性、所需磁盘、预计 bytes/object 数、独占条件；
   输出用户可理解的计划，不改变权威状态。
3. **Backup**：通过 SQLite Backup API/VACUUM INTO 生成 catalog snapshot；需要时生成 CAS
   manifest，而不是复制所有可重建 cache。
4. **Prepare**：取得 maintenance lock，阻止新 run、asset ingest 和第二个进程写入；创建
   resumable job/lease。
5. **Apply**：纯 DB 迁移在单一事务完成；涉及 CAS 的迁移按可重复 batch 执行，先生成/验证对象，
   再事务发布引用，并持久记录 cursor。
6. **Verify**：检查 schema、row count/invariant、foreign key、关键引用和 CAS existence；必要时深度
   hash 抽样或全检。
7. **Commit**：最后更新 migration ledger 和 `user_version`，释放 maintenance lock。
8. **Recovery**：进程中断后依据 job state、cursor 和 checksum 继续或从备份恢复；不靠猜测文件名。

不建议把 down migration 作为主要回滚：数据丢失型 schema 变化往往不可逆。安全回滚单位应是经验证
的 backup set 加旧程序兼容边界。每个 release 必须声明最低可直接升级 storage version；超出跨度时
先走中间版本或离线迁移工具。

## 6. 大规模验收与机械门禁建议

这些是对 Yotta 的测试推论，不是上游事实：

- 规模 fixture：1k/10k workflow、100k/1M resource/variant、100k loose blob、混合 1KB–5GB
  object、至少数十 GB 总量。
- 逐故障点中断：配置 write/sync/replace、CAS staging/publish、DB commit 前后、migration 每个
  batch、backup 发布和 GC mark/sweep。
- 并发：第二实例、运行时 reader、素材导入、保存 workflow、checkpoint、backup、GC 同时竞争。
- 正确性：任何已提交 DB 引用都不能指向 missing blob；GC 不能删除 lease/pin/reachable object；
  config 恢复必须选择可验证完整 generation。
- 性能：启动和列表查询不能随对象目录文件数线性扫描；分页内存应与 page size 相关，不与全库记录
  数相关；大对象 Put 必须流式且内存有固定上界。
- 可运维性：暴露 data-root、各类别 bytes、DB/WAL/freelist、对象数、staging/lease、最后
  backup/fsck/GC/migration 状态；诊断包默认脱敏且不包含 Blob 内容或 secret。

## 7. 需要在仓库审计后再决策的项目

外部资料不足以替代 Yotta 自身事实，以下内容必须结合 R1 inventory 决定：

- 是使用一个 catalog DB，还是把高频 run history 拆为单独 DB；拆库会失去天然单事务，需要明确
  交叉一致性协议。
- Workflow Source 保持独立、可导出的 JSON 事实文件，还是进入 catalog 并由 export 投影；这取决于
  当前 compiler/runtime 的事实边界和用户对可移植性的需求。
- auto-vacuum、checkpoint、busy timeout、连接数和 synchronous 等具体值。
- CAS 分片深度、小对象 pack 阈值、GC 宽限期、软/硬 quota、run/log retention。
- portable mode、用户自定义 data root 和企业重定向目录的支持级别。
- 首个正式 storage schema 的兼容范围；开发期废弃格式无需补一次性 reader，但正式 registry、
  backup/recovery seam 应先于上线建立。

## 8. 主要一手来源索引

- Microsoft：
  [Known Folders](https://learn.microsoft.com/en-us/windows/win32/shell/known-folders)；
  [KNOWNFOLDERID](https://learn.microsoft.com/en-us/windows/win32/shell/knownfolderid)；
  [ReplaceFileW](https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-replacefilew)；
  [MoveFileExW](https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-movefileexw)；
  [FlushFileBuffers](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-flushfilebuffers)；
  [GetDiskFreeSpaceExW](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-getdiskfreespaceexw)；
  [CryptProtectData](https://learn.microsoft.com/en-us/windows/win32/api/dpapi/nf-dpapi-cryptprotectdata)
- Go：
  [`os` package](https://pkg.go.dev/os)；
  [`os` source](https://go.dev/src/os/file.go)；
  [Windows file implementation](https://go.dev/src/os/file_windows.go)
- SQLite：
  [WAL](https://www.sqlite.org/wal.html)；
  [Atomic Commit](https://www.sqlite.org/atomiccommit.html)；
  [Transactions](https://www.sqlite.org/lang_transaction.html)；
  [Backup API](https://www.sqlite.org/backup.html)；
  [VACUUM](https://www.sqlite.org/lang_vacuum.html)；
  [PRAGMA](https://www.sqlite.org/pragma.html)；
  [ALTER TABLE](https://www.sqlite.org/lang_altertable.html)；
  [How To Corrupt](https://www.sqlite.org/howtocorrupt.html)
- 跨平台与 CAS：
  [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/latest/)；
  [Git Objects](https://git-scm.com/book/en/v2/Git-Internals-Git-Objects.html)；
  [git-fsck](https://git-scm.com/docs/git-fsck)；
  [git-gc](https://git-scm.com/docs/git-gc)；
  [OCI Content Descriptors](https://github.com/opencontainers/image-spec/blob/main/descriptor.md)；
  [containerd Content Flow](https://github.com/containerd/containerd/blob/main/docs/content-flow.md)
