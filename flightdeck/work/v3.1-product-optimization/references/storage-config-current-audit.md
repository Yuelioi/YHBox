# 配置与资源持久化现状审计

> 审计基线：`10e803e6`，2026-07-25。本文只记录仓库与当前开发数据样本的事实；
> `bin/data`、`bin/settings.json` 和日志只做只读计数，没有复制内容或修改用户数据。

## 结论

Yotta 现在已有若干质量不错的局部存储实现，但还没有一个应用级持久化模型。主要问题不是 JSON
本身，而是路径、版本、启动、故障、查询、备份与清理分别由各 Store 自行决定：

- 正式 GUI 把配置和数据放在可执行文件旁；日志默认使用进程相对路径。安装、升级、开发启动和 CLI
  因工作目录不同会看到不同的数据集合，MSIX/Program Files 等只读安装位置也不允许这种布局。
- Blob、Workflow Source、Program 和 Run 有目录归属标记；Asset、Snippet、Schedule 没有统一的
  data-root manifest。`workspace-3.1` 到 `workspace` 仍由启动代码直接重命名。
- 多数 Store 在启动时扫描整个目录、读取全部 JSON、建立完整内存 map。Asset 还会启动时哈希校验全部
  引用 Blob；Program 会完整解析全部派生程序；Run 会完整打开全部历史记录。
- 同类损坏得到四种不同结果：整个应用拒绝启动、隔离到 recovery、跳过并警告、删除后重建。
- 写入耐久性不统一。多数新 Store 使用 `durablefs`，Schedule 仍使用固定 `.tmp` 加 `os.Rename`；
  settings 又维护一套重复的原子替换实现。
- Schema、artifact 和 store layout 版本彼此独立是正确方向，但没有应用级 layout manifest 与 migration
  journal；`yotta-versions inventory` 也没有覆盖 Run layout、Asset、Snippet、InputClip 和 Node Package
  registry 等全部持久化契约。
- CAS 已具备配额、哈希校验、pin 和 preview/commit sweep，但 live set 由调用方临时全量汇总，尚无
  持久化引用索引、宽限期、回收代际或跨 Store 一致事务。

因此，不应把所有文件简单塞进一个数据库，也不应继续为每个功能复制一个文件 Store。目标应是：
统一 data-root 与 lifecycle，使用事务型 Catalog 管理可查询元数据和引用，保留文件 CAS 与可移植文档，
把派生缓存、历史和日志放进独立生命周期。

## 当前根目录与组合入口

| 根/对象 | 当前位置 | 当前所有者 | 关键事实 |
|---|---|---|---|
| 应用设置 | `<exe>/settings.json` | `services.App` / `settings.go` | 单个 JSON；损坏或任一校验失败时静默回到全部默认值；无 schema header；包含 UI 偏好以及 AI、HTTP、Application、Automation installation/consent |
| 数据根 | `<exe>/data` | `desktopapp.Run` | GUI 与 CLI 分别计算；GUI 还写入进程环境变量 `YOTTA_DATA_DIR` |
| 日志 | settings 中的 `fileDir`，默认相对路径 `logs` | `LogRuntime` / `LogSink` | 相对当前工作目录；按天 append JSONL；没有保留期、总量配额或压缩 |
| 调试截图 | 开发输出路径 `debug/captures` | `screenshot` | 可重建/可删除诊断数据，但不在统一 cache/diagnostics 根下 |
| 凭据 | Windows Credential Manager | `securestore` | 已与 settings 分开；Logical slot 在 settings，secret 不落普通文件 |

正式安装不能继续依赖 exe 同目录可写。仓库当前也没有一个 `DataRoot` 值对象统一提供 config、content、
cache、diagnostics 和 temp 路径；Store 构造器直接接收任意字符串。

## 持久化对象清单

| 对象族 | 当前路径/格式 | Authority 与加载方式 | 写入/并发 | 版本与损坏策略 | 容量/生命周期风险 |
|---|---|---|---|---|---|
| Workflow Source | `data/workspace/workflows/<id>.json` | 可移植权威文档；启动全读入 map | `durablefs`；进程内 RWMutex；revision CAS | 文档 format/version 1、layout 1；不可读对象进入 `.recovery` | 单文档 1 MiB budget、最多 4096；列表/启动仍为 O(n) 全解码 |
| Workflow recovery | `workflows/.recovery/<digest>.json` | 原始损坏字节的 recovery envelope | 修复后先发布 Source 再删除 envelope | envelope schema 1 | 与备份、隔离、迁移失败没有统一保留策略 |
| Program | `data/workspace/programs/<sha256>.json` | 派生编译缓存；启动全读、按当前 catalog/compiler 校验 | `durablefs`；进程内 RWMutex | artifact version 与 layout 1；失效/损坏直接删除重建 | 最多 16384；启动 O(n) 全解析；无 LRU/字节配额 |
| Run record/journal/state | `data/workspace/runs/<run-id>.json` | 运行历史权威记录；每个文件包含整个 journal/value snapshot；启动全读 | 每次 journal append CAS 后重写整个 Run 文件 | record 1、layout 1；任何坏文件阻断启动 | 最多 65536、单文件 16 MiB；append 为累计 O(n²) 写放大；无归档/保留期/分页索引 |
| Workspace files | `data/workspace/files/**` | 工作流经 capability 访问的用户文件 | 临时文件发布；provider 自己做路径与字节限制 | metadata contract，不是根 layout | 内容类型和保留期与工作流元数据混在同一 workspace 根 |
| Global Asset metadata | `data/templates|clips|macros/<guid>.json` | 权威元数据；三个目录启动全读进一个 map | `durablefs`；RWMutex；revision 仅在内存 | record schema 2；任何坏记录、错目录、旧版本或坏 Blob 阻断启动 | 单记录 1 MiB；启动还会校验所有 Blob；查询先复制全量记录 |
| Asset bytes / Workflow Resource bytes | `data/blobs/<sha256>` | 共享不可变 CAS | 单 writer permit、原子发布、进程内 pin | layout 1 marker；启动扫描大小但不全哈希 | 256 MiB/对象、4 GiB 总量写死在 composition；平铺 64-hex 文件；无持久化引用/回收宽限期 |
| InputClip | Asset kind `clip` 的 Blob，二进制 v3 | Global Asset 元数据引用 Blob | 经 Asset/CAS 提交 | codec version 3，旧版本拒绝 | 大录制与模板共用统一 4 GiB 配额，无法按族观测/限额 |
| Macro | Asset kind `macro` 的 Blob，JSON schema 1 | Global Asset 元数据引用 Blob | 经 Asset/CAS 提交 | schema 1，旧版本拒绝 | 同上；Asset record 与 payload schema 分属不同版本体系 |
| Snippet | `data/snippets/<id>.json` | 权威小文档；启动全读 | `durablefs`；RWMutex | schema 1；坏文件跳过并产生内存 warning | 没有目录 marker、数量/总字节限制；warning 不持久化 |
| Schedule | `data/schedules/<id>.json` | 权威小文档；启动全读 | 固定 `<id>.json.tmp` + `os.Rename`；RWMutex | schema 1；坏文件阻断启动 | 无 `durablefs`、无 directory claim、无数量限制；固定 tmp 不利于异常恢复/多进程 |
| Node Package registry | `data/node-packages/registry.json` + `generations/` | 本机安装与信任权威 | registry 用 `durablefs`；generation staging/rename；进程内锁 | registry 2、manifest/trust/signature 1 | 是独立深模块但未纳入应用级 layout/migration/backup |
| Installation profiles | `settings.json` 内数组 | 本机 AI/HTTP/Application/Automation 安装与 consent | 整个 settings clone/merge/validate/replace | 无 settings schema version；加载时内嵌 consent 修正 | 数量增长会放大整个配置写入；设备身份和 UI 偏好共命运 |
| Secret | Windows Credential Manager | secret 权威 | OS API | target name 隐式版本 | 不应迁入 Catalog 明文；需要备份/迁移时只迁 logical slot，不迁 secret |
| 日志 | `logs/yotta-YYYYMMDD.log` | 诊断记录 | buffered append | 无文件 format/layout 版本 | 无 retention/quota；当前开发样本 12 文件约 4.55 MiB |
| 手工备份 | `data/backups/workflow-sources/*` | 人工操作产生 | 无统一 API | 无根 manifest | 当前不是自动备份系统，不能作为 migration rollback 机制 |

## 当前开发数据样本

样本很小，却已覆盖全部主要矛盾：

| 目录 | 文件数 | 约字节数 |
|---|---:|---:|
| `blobs` | 37 | 229,244 |
| `templates` | 18 | 16,954 |
| `workspace/workflows` | 3 | 57,741 |
| `workspace/programs` | 2 | 331,926 |
| `workspace/runs` | 6 | 285,156 |
| `backups/workflow-sources` | 1 | 70,734 |
| `bin/settings.json` | 1 | 2,198 |
| `bin/logs` | 12 | 4,552,034 |

`settings.json` 当前只有 1 个 Application profile 和 1 个 Automation target，尚未暴露大数组成本。数据目录
总计不足 1 MiB，不能用当前启动速度证明设计可以承受 1000+ Workflow/Asset 或 65k Run。

## 一致性与故障矩阵

| 场景 | 当前结果 | 需要的统一规则 |
|---|---|---|
| settings JSON 截断/字段无效 | 静默丢弃整份配置并使用默认值 | 保留坏文件，显示 recovery；默认值只能替代缺失字段，不能掩盖权威配置损坏 |
| 单个 Source 损坏 | 隔离并继续启动 | 适合用户文档；应由统一 quarantine registry 暴露 |
| 单个 Asset/Schedule/Run 损坏 | 整个应用启动失败 | Catalog 应隔离对象级错误；安全关键 registry 可 fail closed，但不应拖垮无关领域 |
| 单个 Snippet 损坏 | 跳过，warning 只在本次内存中 | warning/quarantine 需要可管理、可删除、可导出 |
| Program 缓存损坏/过期 | 删除后按需重建 | 正确；应明确进入 cache 生命周期而非 content backup |
| Blob 孤儿 | 手工 preview/commit sweep 可删除 | 需要持久化 roots、宽限期、第二次可达性确认和回收审计 |
| 进程在多对象操作中崩溃 | 各文件各自原子，但跨文件可能只完成一半 | Catalog transaction + Blob “先对象、后引用”提交协议 |
| 迁移中崩溃 | 只有 Source 单文档迁移；workspace root 直接 rename | data-root migration journal：preflight、snapshot/backup、step commit、resume/rollback |
| 两个进程打开同一 data-root | 仅各自进程内 mutex；无法互斥 | 根级 single-writer lease；CLI 读写必须通过相同 lease/服务边界 |
| 用户复制 WAL 模式数据库单文件 | 尚未使用 SQLite | 备份必须使用 SQLite backup/VACUUM INTO 或停机 checkpoint；不能普通复制活动主文件 |

## 规模验收基线

后续实现不能只复用当前 67 个文件样本，应至少机械验证：

- 10,000 Global Asset metadata、每个 1–4 variants，列表查询只取当前页且不读 Blob。
- 2,000 Workflow Source、10,000 Program cache、65,536 Run summary；冷启动不全量解析 Run journal。
- 4 GiB CAS、至少 100,000 小对象和若干 256 MiB 对象；启动不全量哈希。
- Run journal 10,000 events；追加写入量近似 O(n)，而非每次重写历史导致 O(n²)。
- 杀进程发生在 Blob 写入、Catalog 引用提交、migration step、backup、GC mark/sweep 的每个边界。
- 两个 GUI/CLI writer 争用同一 data-root 时，第二个获得明确的只读或“已占用”结果，不发生双写。
- settings、单个用户对象、派生 cache、Catalog index 和 Blob 分别损坏时，恢复范围符合其 authority。
