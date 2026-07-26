# R2 — RootSet 与持久化生命周期基座

## Outcome

完成。当前开发数据不兼容、不迁移、不删除；首个正式发布后的升级由独立 layout version、不可变
migration registry、preflight/backup/journal/verify/resume 协议承接。

## Journey

先消除“exe 旁、当前工作目录和各 Store 自己猜路径”的根本不确定性，再引入数据库或搬迁具体对象。
R2 建立所有后续存储改造共享的 composition seam，但不自动删除、覆盖或迁移当前开发数据。

## Scope

- 新增平台 `storage.Roots`/`RootResolver`：
  - Windows 正式默认使用 LocalAppData 的 Yotta 专属目录；
  - CLI、测试、开发和显式便携模式通过同一 override contract 选择根；
  - 不通过 exe 位置、当前工作目录或“发现某个 data 目录”隐式切换模式。
- `root.json` 固定 application identity 与独立 layout version；未知更高版本 fail closed。
- GUI、CLI、settings、logs、captures 和各 Store 只接收 Roots 提供的路径，不再自行拼接。
- settings 改为带 schema version、generation 和 payload checksum 的小型 envelope；损坏时保留 recovery，
  不再静默丢弃整份配置。
- 根级 single-writer lease 阻止两个 GUI/CLI writer 双写；第二实例得到明确错误或只读诊断能力。
- `yotta-versions inventory` 覆盖全部持久化 artifact/schema/layout，而非只列部分 Store。
- 提供只读 health/inventory：物理根、类别 bytes/object count、版本、recovery/staging 状态；输出默认脱敏。

## Non-goals

- 不在 R2 引入 SQLite driver、Content Catalog 或 Run Ledger schema。
- 不迁移 Asset、Workflow、Run、Schedule、Snippet 或 Node Package 数据。
- 不清理 `bin/data`、`bin/settings.json`、相对 `logs` 或任何用户目录。
- 不承诺任意网络盘作为可写正式根；便携自定义根需显式风险校验。

## Acceptance

1. Windows resolver 使用平台本机数据位置，安装目录只读时仍可启动和保存。
2. `task dev` 与测试使用显式隔离根；双击正式 exe 不依赖 cwd。
3. settings 在 write/sync/publish 各 kill-point 后能机械选择最新完整 generation；坏文件可见且可恢复。
4. 第二 writer 不得打开同一根；进程异常退出后 lease 可重新取得。
5. 所有生产路径组合集中在 composition root，仓库门禁拒绝新增 `<exe>/data`、裸相对 `logs` 与
   `workspace-3.1` 类产品版本路径。
6. version inventory 包含 settings envelope、root layout、Blob/Source/Program/Run layout、
   Asset/Schedule/Snippet/Macro/InputClip、Node Package registry 与 portable artifact contracts。
7. `task check`、Windows path/lease/atomic-file 故障注入测试和只读安装位置 smoke 通过。

## Next

进入 R3 Catalog foundation：先锁定 SQLite adapter、Content Catalog/Run Ledger application identity、
schema registry、online backup、quick-check 与 kill-point fixture，不在 Store 构造器中迁移具体领域数据。

## Evidence

- `task check`：持续后台 wrapper 写出同一进程退出码 `0`，30 个受影响 Go 包及版本/bindings 门禁通过。
- `task build`：退出码 `0`，`bin/Yotta.exe` 为 3.1.0 WINDOWS_GUI。
- 修正后的 Windows isolated smoke：`scratch/app` 与 `scratch/profile` 分离，RootSet manifest 存在，
  exe 旁无 `data`/`settings.json`，进程存活 5 秒且清理退出 `0`。
- `bin/Yotta.CLI.exe health`：默认路径脱敏，layout 1 supported，staging/recovery/unknown 均可报告。
