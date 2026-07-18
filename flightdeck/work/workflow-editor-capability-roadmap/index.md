---
topic: workflow-editor-capability-roadmap
title: 3.1 产品能力连续性审计与升级路线
summary: 审计旧产品能力与 3.1 现状，按唯一新架构、用户体验和必要性决定恢复、重做或删除，并完成发布前能力补齐。
---

## State

In progress。3.1 尚未发布；Stage 1–10 的 major upgrade 范围已完成。Stage 11 / Slice 20 已把 Windows 桌面自动化权限模型收敛为默认管理员运行，删除按需提权兼容分支，并保留捕获即安装与当前安装项批量授权；自动门禁、production build 和 exe 内嵌 manifest 验证均已通过，等待真实管理员游戏窗口 smoke 后关闭 Topic。

## Next

使用新构建的 `bin/Yotta.exe` 完成一次真实管理员游戏窗口 smoke：确认启动出现 UAC、F9 可捕获并安装/绑定窗口目标、批量授权后工作流可执行。

## Read now

- work/workflow-editor-capability-roadmap/20-desktop-target-uac-and-consent-ux.md
- work/workflow-editor-capability-roadmap/upgrade-plan.md

## Read if

- work/workflow-editor-capability-roadmap/slices/map.md — 查询各 Slice 最终状态
- work/workflow-editor-capability-roadmap/artifacts/legacy-product-capability-diff.md — 对照旧能力取舍
- work/workflow-editor-capability-roadmap/capability-audit.md — 查询恢复范围和发布阈值
- knowledge/architecture/content-addressed-workflow-artifacts.md — 修改 Workflow/Node durable identity
- knowledge/build/build.md — 进入发布、打包或真机 smoke
- knowledge/input/windows-uac-window-capture.md — 修改 Windows 权限、窗口捕获、输入注入或自启策略

## Progress

- Stage 1–3 恢复可靠图编辑、类型感知连线、选择/布局、诊断/真调试、模板节点、Blob 预览与键鼠/轨迹录制。
- Stage 4 关闭暗色、端口、alert/toast、Start/Delete、桌面安装/F9、工作流库、AI endpoint 与 launcher 回归。
- Stage 5–6 完成平台中立 Adapter seam 和 Android/ADB 安装、创作、运行闭环。
- Stage 8–10 完成 Workflow Source portability、资产规模化/安全清理、Browser CDP exact installation 与 Source-native 多图闭环。
- Stage 11 将 Windows manifest 固定为 `requireAdministrator`，删除 Tools/desktopapp/frontend 的按需提权 API 与 UI。
- Windows 自启改为当前交互用户的 `ONLOGON + /IT + /RL HIGHEST` 计划任务；Yotta 启动的桌面应用按产品约定继承管理员权限。
- Stage 11 验收：`task check` 通过（global coverage 65.2%，38 files / 152 frontend tests）；`task build` 通过，并从 `bin/Yotta.exe` 提取确认内嵌 `requireAdministrator` manifest。

## Open questions

- Slice 20 仍需在真实管理员游戏窗口上确认默认提权后的 F9 捕获、捕获即安装/绑定和批量授权闭环；自动测试不能替代 UAC prompt、反作弊与目标进程完整性级别的宿主 smoke。
- 删除工作流时，关联 schedule/launcher item 继续采用引用阻止；历史 Run journal 默认保留，除非后续产品需求明确改变。
- 自定义 AI 地址首期保持 provider-native Responses/Messages，不静默兼容 Chat Completions。
- 对外发布前仍需按 release 流程决定签名、安装包与许可证表述；当前 LICENSE 是 source-available，不能称 OSI open source。
