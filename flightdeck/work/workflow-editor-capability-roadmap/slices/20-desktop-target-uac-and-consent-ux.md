---
slice: "20"
title: 桌面目标 UAC、捕获安装与批量授权
status: blocked
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

真实管理员游戏窗口 smoke 仍需要用户在宿主桌面完成 F9 捕获和目标工作流运行。清除不兼容的旧开发 `bin/data` 后，UAC 启动已通过并保持运行；3.1 尚未发布，不为旧开发数据引入迁移层。

## Verification

自动化证据（2026-07-18）：

- `task check` 通过：162 frontend tests、Go 65.5% coverage/vet/staticcheck、167 Wails models、production build 与 bundle budget。
- `task build` 通过，新 `bin/Yotta.exe` 为 31,314,944 bytes。
- 用 Windows Kits `mt.exe` 提取新构建的资源 #1，确认 `requestedExecutionLevel level="requireAdministrator" uiAccess="false"`。

已完成人工启动：

- 双击新 `bin/Yotta.exe` 后出现 UAC；清除旧开发数据后应用正常进入主界面，随后再次启动保持运行（PID 18516）。
- 对照结果：同一构建保留旧 `bin/data` 时在日志初始化前退出，删除旧数据后启动成功；manifest/UAC 不是故障原因。

仍需人工：

1. 打开自动化目标捕获，切换到管理员运行的“异环”，按 F9；确认能捕获 exact title/class。
2. 若应用未安装，确认弹窗摘要中的 exe 路径正确；一次确认后同时创建 application 与 window target，不残留旧应用默认值。
3. 使用“授权当前全部”完成 application/target 授权；确认没有逐项重复弹窗，也没有把未来新增项静默授权。
4. 创建最小目标工作流，至少执行一次截图、一次键盘输入和一次鼠标输入；确认异环收到输入，截图非黑屏且窗口身份未漂移。
5. 退出重开后再次绑定/运行，确认 UAC、目标身份与授权行为稳定。

## Out of scope

`asInvoker` 兼容模式、按需提权、双 run level、独立提权 helper、旧 HKCU Run 迁移、静默授权未来安装项、绕过 workflow capability/admission/arm、规避第三方反作弊或 Windows secure desktop。

## Result

Implementation complete, host smoke pending。

- Windows manifest 固定为 `requireAdministrator`；启动权限测试改为约束该发布契约。
- 删除 Tools service、desktopapp 和 Settings Automation 中的提权状态、按需 runas 与重启 UI。
- Windows 自启改为 `schtasks /Create /SC ONLOGON /IT /RL HIGHEST`，禁用时按任务名幂等删除，并覆盖参数、删除与不安全路径测试。
- 捕获未安装应用时继续以共享确认弹窗展示精确路径，并在一次 Settings mutation 中写入应用与窗口目标。
- Automation service 继续原子批量 seal/revoke 当前应用与目标 consent digest；任何后续身份变化仍按现有摘要规则失效。
- 代码、完整门禁、production build、内嵌 manifest 与 UAC clean-data 启动均已完成；只等待上述真实管理员窗口 smoke。
