# Headless CLI reference

正式 Windows 构建会生成 `bin/Yotta.CLI.exe`。它打开与桌面相同的 `internal/localruntime`、Source Store、
Compiler、Application、Run Ledger、provider 和 Configured Target generation，不是简化版或第二套 runtime。

## Workflow commands

```powershell
bin/Yotta.CLI.exe [--data-root <directory>] [--timeout 5m] validate <source.json>
bin/Yotta.CLI.exe [--data-root <directory>] [--timeout 5m] compile <workflow-id>
bin/Yotta.CLI.exe [--data-root <directory>] [--timeout 5m] inspect <workflow-id>
bin/Yotta.CLI.exe [--data-root <directory>] [--timeout 5m] [--principal local-cli] run <workflow-id>
```

- `validate` 对文件中的 draft Source 运行当前 strict parser/compiler，不导入它。
- `compile` 编译当前 profile 中的 Source，并输出 source/program hash 与 diagnostics。
- `inspect` 输出 workflow ID、revision、source hash 和 canonical Source。
- `run` 持久化并等待一次正式 Run；成功或终止时输出 Run Record JSON。

CLI 使用 Go `flag.FlagSet`；所有 flag 必须写在 command 前，`run wf --timeout 5m` 不会被当作 flag 解析。
通用参数：

```powershell
--data-root <directory>   # 显式选择 profile；否则使用 YOTTA_ROOT 或平台默认目录
--timeout 5m             # workflow command deadline
--principal local-cli    # Run principal
```

Profile 有单 writer lease。不要同时让 GUI 和 CLI 以写模式打开同一个 profile；自动化脚本应为隔离运行显式
提供 `--data-root`。`validate`、`compile`、`inspect` 和 `run` 都会先执行已登记的 profile migration，再打开完整
local runtime；`validate` 只是不把传入 Source 导入 Catalog，并不代表它以只读方式打开 profile。

## Health

```powershell
bin/Yotta.CLI.exe [--data-root <directory>] health
bin/Yotta.CLI.exe [--data-root <directory>] --show-path health
```

Health 输出 storage root identity/layout 与 Content Catalog/Run Ledger 检查。默认隐藏物理路径；只有显式
`--show-path` 才回显。该命令检查现状，不执行 migration。

## Storage migration

```powershell
bin/Yotta.CLI.exe [--data-root <directory>] migrate plan
bin/Yotta.CLI.exe [--data-root <directory>] migrate apply
bin/Yotta.CLI.exe [--data-root <directory>] migrate resume
bin/Yotta.CLI.exe [--data-root <directory>] migrate rollback
bin/Yotta.CLI.exe [--data-root <directory>] migrate list
bin/Yotta.CLI.exe [--data-root <directory>] migrate quarantine <legacy-run-file>
bin/Yotta.CLI.exe [--data-root <directory>] migrate restore <record-name>
bin/Yotta.CLI.exe [--data-root <directory>] migrate export <destination>
```

Migration 命令只支持 `internal/storage/migrate` 注册的相邻 layout 路径，并保留 journal、backup 与 quarantine
证据。运行前先关闭使用同一 profile 的桌面进程；不要把这些命令当任意旧格式转换器。

CLI 的精确参数仍以 `cmd/yotta/main.go` 为权威；正式构建和发布 payload 见 [Releasing Yotta](../../RELEASING.md)。
