---
slice: "09"
title: 桌面应用选择与窗口目标安装
status: completed
---

# Slice 9：桌面应用选择与窗口目标安装

## Outcome / Question

桌面应用文件选择可以无副作用取消；新建窗口目标从空白安装草稿开始，并恢复 F9 一键捕获和绑定 exact 窗口，不要求用户手填易错的标题与类名。

## Completion criterion

- 真机复现并区分 Wails 文件选择取消、空 path、对话框错误和 executable 检查失败；取消必须关闭 pending 状态并成为干净 no-op，不重开、不报错、不 toast、不创建记录。
- 空状态文案和按钮不表现成无法退出的强制安装流程；重复点击只产生一个文件对话框。
- addTarget 不再读取 applications[0] 作为隐式默认，也不在草稿不完整时 commit；“新建”与“复制已有目标”是两个显式动作。
- 设置页提供“捕获窗口”按钮并支持 F9，调用现有 StartWin32WindowTargetCapture；捕获期间可取消、超时并避免重复注册热键。
- 捕获结果按 executable identity 匹配已安装 Application，并回填 exact title/class；未安装、零匹配、多匹配和应用摘要变化都有可操作诊断。
- 保存后的 target 继续遵守 exact installation、固定 backend/timeout、consent invalidation 和不把 PID/HWND 写入图的边界。
- 所有确认和错误使用 Nuxt UI；成功保存原地反馈，不发成功 toast。

## Blocked by

Stage 3 批量验收；实现前读取 installed-input-authority 与 Settings UI 约定。

## Verification

先做 picker cancellation、draft lifecycle、capture mapping 和 settings store 的定向测试；Stage 4 完成后统一运行 task check、task build、正式 Windows 文件对话框/F9/WebView smoke 与人工视觉检查。

## Out of scope

模糊窗口匹配、active-window 隐式运行目标、把 HWND/PID 持久化、自动授予 workflow consent、Android 设备安装。

## Result

Completed。文件选择取消现在会结束 pending 并显示可再次选择的行内空状态，不创建记录、不重开、不 toast；picker/inspect 异常行内报错并由 finally 收口 busy 状态。新目标从 applicationSlot/title/class 全空草稿开始，不完整时不 commit；复制是独立动作且移除 consent。设置页已接回 Start/CancelWin32WindowTargetCapture、30 秒超时和 F9 事件，事件只暴露 executable/title/class；前端重新检查可执行文件并按规范化完整路径 + SHA-256 唯一匹配安装记录，零匹配和多匹配都有可操作诊断，成功原地反馈。Go tools/winutil、3 个前端文件 6 tests、1517-key i18n、typecheck 与 diff check 均通过；正式文件对话框/F9 真机 smoke 留到 Stage 4 批量门禁。
