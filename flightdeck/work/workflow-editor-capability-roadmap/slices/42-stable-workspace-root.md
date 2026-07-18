---
slice: "42"
title: 稳定 Workspace 根目录
status: completed
---

# Slice 42：稳定 Workspace 根目录

## Outcome / Question

移除 production composition 中的版本化数据根 `workspace-3.1`，让产品升级不再通过目录名制造新工作区或测试感。

## Completion criterion

- canonical 根为 `<dataRoot>/workspace`。
- 仅旧目录存在时同父目录执行一次性、可恢复迁移，保留 sources/programs/runs 与原子语义。
- 新旧同时存在时不静默合并或任选，返回可操作诊断。
- clean/current/legacy/conflict/corrupt 矩阵有测试；smoke/docs/knowledge 不再硬编码旧名。
- 不引入第二条兼容 runtime；迁移后只运行 stable root。

## Verification

- appbootstrap migration 单元/集成测试。
- production build 临时 data root 的首次启动、迁移和重启测试。
- 真机确认现有 `bin/data/workspace-3.1` 可迁移且工作流仍可见。

## Out of scope

- 不自动合并两个都有数据的 workspace。
- 不在测试通过前直接移动用户真机目录。

## Result

Completed。

- production canonical root 改为 <dataRoot>/workspace；仅 legacy workspace-3.1 存在时使用同父目录原子 rename，一旦新旧并存或任一根是文件即返回诊断。
- appbootstrap clean/current/legacy/conflict/invalid-root 测试通过，build/smoke 路径已切换到稳定根。
- 真实 bin/data 迁移成功：workspace-3.1 已不存在，workspace 存在，原 workflow 03b564f0-cfce-4127-97a2-e0f3973ccc98.json 仍可见。
- production build、重新启动的 UAC Yotta、WebView smoke 和 task check 通过。
