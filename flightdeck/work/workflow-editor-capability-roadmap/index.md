---
topic: workflow-editor-capability-roadmap
title: 3.1 产品能力连续性审计与升级路线
summary: 审计旧产品能力与 3.1 现状，按唯一新架构、用户体验和必要性决定恢复、重做或删除，并完成发布前能力补齐。
---

## State

In progress。3.1 尚未发布；Stage 1–10 的 major upgrade 范围已完成，但真实管理员游戏窗口暴露出桌面目标安装体验仍有 UAC 权限边界、捕获即安装和逐项授权摩擦。Stage 11 / Slice 20 已完成实现与自动门禁，等待使用管理员目标窗口完成一次人工 smoke 后重新关闭 Topic。

## Next

使用新构建的 `bin/Yotta.exe` 验证：普通权限捕获超时可提示并按需管理员重启，管理员实例可捕获真实游戏窗口，未安装应用可在捕获流程中直接安装并绑定。

## Read now

- work/workflow-editor-capability-roadmap/20-desktop-target-uac-and-consent-ux.md
- work/workflow-editor-capability-roadmap/upgrade-plan.md

## Read if

- work/workflow-editor-capability-roadmap/slices/map.md — 查询各 Slice 最终状态
- work/workflow-editor-capability-roadmap/artifacts/legacy-product-capability-diff.md — 对照旧能力取舍
- work/workflow-editor-capability-roadmap/capability-audit.md — 查询恢复范围和发布阈值
- knowledge/architecture/content-addressed-workflow-artifacts.md — 修改 Workflow/Node durable identity
- knowledge/build/build.md — 进入发布、打包或真机 smoke
- knowledge/input/windows-uac-window-capture.md — 排查普通应用能捕获而管理员目标不能捕获

## Progress

- Stage 1–3 恢复可靠图编辑、类型感知连线、选择/布局、诊断/真调试、模板节点、Blob 预览与键鼠/轨迹录制。
- Stage 4 关闭暗色、端口、alert/toast、Start/Delete、桌面安装/F9、工作流库、AI endpoint 与 launcher 回归。
- Stage 5–6 完成平台中立 Adapter seam 和 Android/ADB 安装、创作、运行闭环。
- Stage 8–9 完成 Workflow Source portability、资产规模化/安全清理，以及 Browser CDP exact installation 产品闭环。
- Stage 10 完成 Source-native 多图 schema、authoring、compiler/runtime/debugger、编辑器与 MCP 闭环。
- Stage 11 已实现 UAC 感知捕获超时、按需管理员重启、捕获即安装/绑定，以及当前应用与自动化目标的原子批量授权/撤销。
- Stage 11 自动门禁：完整 `task check` 通过（global coverage 65.1%，38 files / 152 frontend tests）；production `task build` 通过并更新 `bin/Yotta.exe`。

## Open questions

- Slice 20 仍需在真实管理员游戏窗口上确认 UAC 重启后的 F9 捕获闭环；自动测试不能替代 UAC prompt 与目标进程完整性级别的宿主 smoke。
- 删除工作流时，关联 schedule/launcher item 继续采用引用阻止；历史 Run journal 默认保留，除非后续产品需求明确改变。
- 自定义 AI 地址首期保持 provider-native Responses/Messages，不静默兼容 Chat Completions。
- 对外发布前仍需按 release 流程决定签名、安装包与许可证表述；当前 LICENSE 是 source-available，不能称 OSI open source。
