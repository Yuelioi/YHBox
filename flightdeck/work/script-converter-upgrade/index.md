# Script Converter Upgrade

STATUS: implemented, pending broader conversion smoke

## Goal

升级“子图转脚本”转换器，让复杂流程优先落到 Script 节点表达，而不是被 `Loop` / `Expr` 等结构拒转。

## Current Scope

- 支持 `Expr` 纯数据节点转成内联 JS 表达式。
- 支持 `Loop` 的 `count` / `forever` 模式转成 JS `for` / `while`。
- 支持 `Break` / `Continue` 转成 JS `break` / `continue`。
- 支持 exec 分支汇合：转换时把汇合后的尾部复制进每个 JS 分支。
- 不改 Script runtime；goja JS 已支持循环、条件、异常和表达式。

## Notes

- 首批改动只落在前端转换器和测试。
- 已通过 red-green 覆盖 `Expr`、`Loop count/forever`、`Break`、`Continue`、条件分支汇合。
- fishing-v2 暂不重写为脚本，等转换器能力稳定后再选择局部替换。

## Verification

- `pnpm --dir frontend test -- subgraphToScript`
- `pnpm --dir frontend typecheck`
