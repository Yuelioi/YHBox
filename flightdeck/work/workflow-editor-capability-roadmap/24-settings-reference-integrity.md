# Slice 24 — Settings 引用完整性

## Outcome

已完成。删除 desktop application 时，Settings service 在同一次 merge 中清理引用该 application slot 的 automation targets；保存后不会产生 validator 才发现的悬空 target。

## Evidence

`internal/services/settings_test.go` 覆盖删除 `htgame` 时同时删除 `window-target`，并保留未删除 application 的无关 targets。相关 services 测试和 2026-07-18 完整 `task check` 通过。
