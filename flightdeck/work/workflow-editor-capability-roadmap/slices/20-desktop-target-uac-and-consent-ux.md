---
slice: "20"
title: 桌面目标 UAC、捕获安装与批量授权
status: in_progress
---

# Slice 20：桌面目标 UAC、捕获安装与批量授权

## Outcome / Question

让 Windows 桌面自动化采用单一、可预测的管理员权限模型，并把“捕获窗口、安装应用、绑定目标、授权使用”从跨页面逐项操作收敛为连贯流程。

## Completion criterion

- Windows production manifest 固定为 `requireAdministrator`；Yotta 每次启动经 UAC 进入管理员完整性级别。
- 删除普通权限、按需 runas、提权状态探测和双运行级别兼容分支；Yotta 启动的桌面应用按产品约定继承管理员权限。
- Windows 自启使用当前交互用户的最高权限登录计划任务，不使用无法满足默认管理员模型的 HKCU Run。
- 没有预装桌面应用时仍可新建 Windows 目标；捕获后可显式确认，原子安装摘要固定的应用并绑定 exact title/class。
- 用户可一次授权或撤销当前全部桌面应用与自动化目标；后续新增或身份变化仍需新授权，不绕过 capability、admission 或 arm。
- 完整门禁、production build、内嵌 manifest 检查和真实管理员窗口 smoke 通过。

## Blocked by

真实管理员游戏窗口 smoke 需要用户在宿主桌面完成 UAC prompt、F9 捕获和目标工作流运行。

## Verification

- `task check`
- `task build`
- `mt.exe -inputresource:bin\Yotta.exe;#1` 确认内嵌 `requireAdministrator`
- 人工：UAC 启动 Yotta → F9 捕获管理员游戏 → 未安装应用确认安装并绑定 → 批量授权 → 运行目标工作流。

## Out of scope

`asInvoker` 兼容模式、按需提权、双 run level、独立提权 helper、旧 HKCU Run 迁移、静默授权未来安装项、绕过 workflow capability/admission/arm、规避第三方反作弊或 Windows secure desktop。

## Result

Implementation complete, host smoke pending。

- Windows manifest 固定为 `requireAdministrator`；启动权限测试改为约束该发布契约。
- 删除 Tools service、desktopapp 和 Settings Automation 中的提权状态、按需 runas 与重启 UI；Wails contract 收敛为 14 services / 119 methods / 166 models。
- Windows 自启改为 `schtasks /Create /SC ONLOGON /IT /RL HIGHEST`，禁用时按任务名幂等删除，并覆盖参数、删除与不安全路径测试。
- 捕获未安装应用时继续以共享确认弹窗展示精确路径，并在一次 Settings mutation 中写入应用与窗口目标。
- Automation service 继续原子批量 seal/revoke 当前应用与目标 consent digest；任何后续身份变化仍按现有摘要规则失效。
- `task check` 通过：global coverage 65.2%，38 个前端测试文件 / 152 项测试全绿。
- `task build` 通过；从新 `bin/Yotta.exe` 提取的 manifest 已确认 `requestedExecutionLevel level="requireAdministrator"`。
