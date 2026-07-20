# Slice 24 — Settings 引用完整性

## Outcome / Question

删除 desktop application 时能否在同一领域事务中处理所有 automation target 引用，避免保存悬空设置？

## Completion criterion

删除 application slot 会原子清理引用它的 targets，同时保留无关 applications 和 targets。

## Blocked by

无。

## Verification

`internal/services/settings_test.go` 覆盖删除 `htgame` 同时删除 `window-target`，并保留无关 target；相关 services 测试和 2026-07-18 完整 `task check`。

## Out of scope

桌面目标真实管理员游戏窗口 smoke，由 Slice 20 承担。

## Result

Completed。Settings service 在同一次 merge 中清理依赖 target，保存后不再产生 validator 才发现的悬空引用。
