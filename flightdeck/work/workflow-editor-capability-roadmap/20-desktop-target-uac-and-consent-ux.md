---
slice: "20"
title: 桌面目标 UAC、捕获安装与批量授权
status: in_progress
---

# Slice 20：桌面目标 UAC、捕获安装与批量授权

## Outcome / Question

让 Windows 桌面目标安装在管理员应用场景中给出可执行的 UAC 处理路径，并把“捕获窗口、安装应用、绑定目标、授权使用”从跨页面逐项操作收敛为连贯流程。

## Completion criterion

- Yotta 保持默认 `asInvoker`，不强制每次启动弹 UAC。
- 捕获超时时区分普通权限与已提权实例；普通权限实例可经显式确认按需管理员重启。
- 没有预装桌面应用时仍可新建 Windows 目标；捕获后可显式确认，原子安装摘要固定的应用并绑定 exact title/class。
- 用户可一次授权或撤销当前全部桌面应用与自动化目标；后续新增或身份变化仍需新授权，不绕过 capability、admission 或 arm。
- 完整门禁、production build 和真实管理员窗口 smoke 通过。

## Blocked by

真实管理员游戏窗口 smoke 需要用户在宿主桌面完成 UAC prompt 与 F9 操作。

## Verification

- `task check`
- `task build`
- 人工：普通权限捕获管理员游戏 → 超时提示 → 管理员重启 → F9 捕获 → 未安装应用确认安装并绑定 → 批量授权/撤销。

## Out of scope

把 manifest 改成 `requireAdministrator`、静默提权、永久自动授权未来安装项、绕过 workflow capability/admission/arm、规避第三方反作弊保护。

## Result

Implementation complete, host smoke pending。

- Tools service 暴露当前提权状态和 `runas` 管理员重启；新实例启动成功后旧实例走 Wails 正常退出。
- Settings Automation 在 UAC 超时处提供原地说明和重启入口。
- 捕获未安装应用时以共享确认弹窗展示精确路径，并在一次 Settings mutation 中写入应用与窗口目标。
- Automation service 原子批量 seal/revoke 当前应用与目标 consent digest；任何后续身份变化仍按现有摘要规则失效。
- `task check` 通过：global coverage 65.1%，38 个前端测试文件 / 152 项测试全绿，Wails contract 14 services / 121 methods / 166 models。
- `task build` 通过，新的 `bin/Yotta.exe` 已生成。
