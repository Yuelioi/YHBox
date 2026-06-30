# Script Converter Upgrade

STATUS: implemented, fishing-v2 single-script locally, pending game smoke

## Goal

升级“子图转脚本”转换器，让复杂流程优先落到 Script 节点表达，而不是被 `Loop` / `Expr` 等结构拒转。

## Current Scope

- 支持 `Expr` 纯数据节点转成内联 JS 表达式。
- 支持 `Loop` 的 `count` / `forever` 模式转成 JS `for` / `while`。
- 支持 `Break` / `Continue` 转成 JS `break` / `continue`。
- 支持 exec 分支汇合：转换时把汇合后的尾部复制进每个 JS 分支。
- 纯数据节点按引用点内联，不再跨 block 复用 `const vN`，避免循环内声明被循环外引用。
- 全量扫描 `bin/data/subgraphs`: 20/20 可转，无剩余拒转项。
- 不改 Script runtime；goja JS 已支持循环、条件、异常和表达式。

## Notes

- 首批改动落在前端转换器和测试；后续补了 Script 依赖扫描器，防止 `Subgraph({SubgraphID:"sg-uuid"})` 的内层 UUID 被误算成 template。
- 已通过 red-green 覆盖 `Expr`、`Loop count/forever`、`Break`、`Continue`、条件分支汇合。
- fishing-v2 已从“顶层图调用 12 个脚本化子图”继续压平为“Start → Win32WindowTarget → 单主 Script → Stop”。主脚本内含状态函数和 helper 函数；原始子图备份在 `bin/data/_backups/fishing-v2-subgraphs-before-script-mode/`，压平前 graph 备份在 `bin/data/_backups/fishing-v2-graph-before-single-script.json`。
- 已删除当前全局池里的 12 个 fishing-v2 子图文件；`bin/data/containers/fishing-v2/yotta-lock.json` 已刷新，dependencies.subgraphs 为空，templates 仍为真实 18 个 template GUID。
- 压平时修正了两处脚本化语义：`Switch({Value})` 在脚本里没有 `cases` 配置，改用本地 JS helper 保留 phase/right/left/default 语义；`DualColorBarTrack` 的 `_barOuterX/_barInnerX/_barOuterW` 捕获改成直接读返回数据 `OuterX/InnerX/OuterWidth`。
- `bin/` 被 gitignore，本地容器数据变更不会进入提交；仍需游戏内 smoke 没饵/没钱/关闭自动卖鱼分支。

## Verification

- `pnpm --dir frontend test -- subgraphToScript`
- `pnpm --dir frontend typecheck`
- `go test ./internal/services/script`
- `node -e ...` compiled fishing-v2 single Script body; graph=4 nodes/3 edges, no `Subgraph(`, subgraphDeps=[]
