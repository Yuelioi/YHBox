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

真实管理员游戏窗口 smoke 仍需要在宿主桌面完成录制、鼠标与组合键运行。清除不兼容的旧开发 `bin/data` 后，UAC 启动已通过；3.1 尚未发布，不为旧开发数据引入迁移层。

首次人工捕获已经暴露并定位一个启动快照架构缺陷：目标成功写入 Settings 后，当前进程的 recording `AuthoringTargets` 与 Workflow admission 仍是 composition root 启动时安装的不可变快照，因此同一进程分别报告 `RECORDING_TARGET_UNAVAILABLE` 与 `admission.target_unavailable`。这不是目标身份、consent 或鼠标参数失败。该设计不能靠重启提示掩盖，修复由 Slice 26 负责：同一进程原子切换 authoring、admission 与 provider generation。

重启后的第二次 smoke 进一步定位到 3.1 窗口身份回归：异环真实 Win32 title 为 `"异环  "`（两个尾随空格）、class 为 `UnrealWindow`，可执行文件 digest 完全一致；Settings 捕获链却把 title 裁成 `"异环"`，同时 installed resolver 写死逐字符精确匹配，导致 recording、截图、鼠标与按键共用的 target resolver 全部失败。旧版实际支持 `TitleMatch=exact|regex`，底层 `winutil.MatchSpec` 也仍保留该能力；3.1 不能把它静默阉割。

## Verification

自动化证据（2026-07-18）：

- 最新 `task check` 通过：168 frontend tests、Go 65.5% coverage/vet/staticcheck、Wails 14 services / 119 methods / 167 models、production build 与 bundle budget。
- `task build` 通过，新 `bin/Yotta.exe` 为 31,314,944 bytes。
- 用 Windows Kits `mt.exe` 提取新构建的资源 #1，确认 `requestedExecutionLevel level="requireAdministrator" uiAccess="false"`。

已完成人工启动：

- 双击新 `bin/Yotta.exe` 后出现 UAC；清除旧开发数据后应用正常进入主界面，随后再次启动保持运行（PID 18516）。
- 对照结果：同一构建保留旧 `bin/data` 时在日志初始化前退出，删除旧数据后启动成功；manifest/UAC 不是故障原因。

仍需人工：

1. 用新构建重启 Yotta，确认已持久化的 `window-target` 被 recording 与 Workflow runtime 同时安装。
2. 重新捕获异环，确认标题原样保留尾随空格、捕获后模式为 exact；另切换 regex 并验证 `^异环\s*$` 可保存、无效正则被拒绝。
3. 从资源库或编辑器开始一次简易录制，完成鼠标点击与按键，确认 HUD、预览和保存链路可用。
4. 运行“Run 开始 → 激活目标 → 点击指针”，确认默认 50%/50%、left、50 ms 能在异环收到点击。
5. 添加“按下组合键”，点击 `keys` 的录制控件录入一个无破坏组合键并运行，确认无需编辑 JSON 且目标收到按键。
6. 至少执行一次截图，确认非黑屏且窗口身份未漂移；退出重开后重复一次目标运行，确认 UAC、目标身份与授权稳定。

## Out of scope

`asInvoker` 兼容模式、按需提权、双 run level、独立提权 helper、旧 HKCU Run 迁移、静默授权未来安装项、绕过 workflow capability/admission/arm、规避第三方反作弊或 Windows secure desktop。

## Result

Implementation complete, host smoke pending。

- Windows manifest 固定为 `requireAdministrator`；启动权限测试改为约束该发布契约。
- 删除 Tools service、desktopapp 和 Settings Automation 中的提权状态、按需 runas 与重启 UI。
- Windows 自启改为 `schtasks /Create /SC ONLOGON /IT /RL HIGHEST`，禁用时按任务名幂等删除，并覆盖参数、删除与不安全路径测试。
- 捕获未安装应用时继续以共享确认弹窗展示精确路径，并在一次 Settings mutation 中写入应用与窗口目标。
- Automation service 继续原子批量 seal/revoke 当前应用与目标 consent digest；任何后续身份变化仍按现有摘要规则失效。
- 捕获成功反馈现在显式标注“重启后生效”，说明 recording 与 Workflow runtime 都需重新装载目标；录制与 admission 的目标不可用错误也给出同一恢复动作。
- Run 时间线和编辑器 toast 不再直接显示 `admission.target_unavailable` 裸码，而是走统一本地化错误消息。
- `List<KeyCode>` 输入新增契约感知的组合键录制器，暂停全局热键后捕获并规范化 `CTRL/SHIFT/ALT + KeyCode`，不再让用户编辑 JSON。
- Windows target profile、Settings、Wails contract、UI 与 installed resolver 恢复 `windowTitleMatch=exact|regex`；capture 固定 exact 并逐字符保留原始 Win32 title（包括首尾空格），regex 使用 Go RE2 且在 seal 阶段验证。cached HWND 复验与重新枚举共享同一 selector。
- 代码、完整门禁、production build、内嵌 manifest 与 UAC clean-data 启动均已完成；只等待新 exe 的上述真实管理员窗口 smoke。
