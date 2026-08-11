# Local storage

## Profile root

`internal/storage.Resolve` 是正式路径的唯一解析入口，优先级为：显式 `StorageRoot` / CLI `--data-root`，
其次 `YOTTA_ROOT`，最后平台默认目录。Windows 默认值是：

```text
%LOCALAPPDATA%\Yotta\Yotta
```

Linux/macOS 预览 host 使用 `os.UserConfigDir()` 下的 `Yotta/Yotta`。`task dev` 使用仓库内隔离 profile；
exe 目录、当前工作目录、根目录的 `data/` 或 `settings.json` 都不是生产数据位置。

profile 由 `root.json` 标识为 `yotta.storage-root`。进程持有 `runtime/writer.lock`，同一 profile 同一时间只能
有一个 writer。未知 identity/layout、非空未认领目录或未来 schema 都 fail closed。当前 root/database/store
版本不在本页复制，使用 `task versions:inventory` 从所属 package 聚合读取。

## 当前数据地图

| 路径（相对 profile） | 内容与 authority |
| --- | --- |
| `root.json` | profile identity 与 root layout；由 `internal/storage` 拥有 |
| `config/settings.json` | versioned settings generation envelope；同目录有 `.bak`、staging 和 `recovery/` |
| `catalog/content.db` | Content Catalog；Workflow Source、quarantine/release/install、asset、Blob 引用与 GC metadata |
| `state/runs.db` | Run Ledger；Run、event、value/attempt/action 等运行事实 |
| `objects/sha256/` | immutable content-addressed Blob bytes；metadata 与引用在 Content Catalog |
| `cache/programs/` | versioned content-addressed Program cache；可删除并从 Source 重建，不进入备份 authority |
| `data/workspace/files/` | Workflow file capability 的受限工作区 |
| `data/schedules/`、`data/snippets/` | 小型文件 Store，由对应 service 管理 |
| `packages/node/` | 已验证 Node Package generations 和 registry authority |
| `documents/exports/` | profile 投影保留的 export 目录；当前 Workflow/Run export service 使用用户显式选择的 destination |
| `diagnostics/logs/`、`diagnostics/crashes/`、`diagnostics/captures/` | 可诊断输出，不是业务 authority |
| `backups/migrations/` | storage migration 的一致备份、journal 与恢复材料 |
| `runtime/`、`tmp/` | writer lease 与临时内容，不是持久业务数据 |

AI API key 不写入 settings、SQLite、日志或 RPC 返回。Windows 使用 Generic Credential 写入 Credential
Manager；非 Windows 预览 host 当前返回 secure store unavailable。代码入口是 `internal/securestore/` 和
`internal/services/ai_secrets.go`。

## 一致性与恢复

- Content Catalog 和 Run Ledger 由 `internal/storage/catalog.Foundation` 独占连接与 migration；service 不
  暴露 `*sql.DB` 或拼接跨库 transaction。两库启用 foreign keys、WAL、busy timeout 和 `synchronous=FULL`。
- Workflow Source、settings、package registry 和其它文件 authority 使用 strict identity/version、
  canonical bytes、CAS/generation 与 durable publish；rename 已提交但目录 sync 不确定时返回 committed warning，
  不能生成第二个 identity。
- settings 从 primary、backup、staging 选择最高有效 generation；冲突或全部损坏进入 recovery-required，
  不用默认值覆盖已有事实。
- 活跃 WAL 数据库不能靠复制单个 `.db` 文件备份。Catalog backup 使用 SQLite Online Backup API，两个快照
  验证完成后最后发布 manifest。
- Program cache、diagnostics、runtime 和 tmp 可按所属模块规则重建/清理；不要把 `objects/sha256` 当无引用
  文件夹手工删除，Blob GC 需要 Catalog 引用和 lease 快照。

诊断 profile 使用 `yotta health`；默认隐藏物理根，只有显式 `--show-path` 才输出。修改 root layout、Catalog/
Run migration、backup、journal 或 recovery 时按[构建与验证指南](../../flightdeck/knowledge/build/build.md)运行
storage migration smoke。
