---
slice: "08"
title: Android 目标连续性与创作模板
status: completed
---

# Slice 8：Android 目标连续性与创作模板

## Outcome / Question

恢复 Android/ADB 在 3.1 唯一 Workflow Source、Catalog、compiler、provider 与编辑器上的完整产品闭环，并决定平台模板是创作引导还是工作流硬分类。

## Completion criterion

- 明确记录产品决策：Windows / Android 创建入口是否只是模板；默认不得阻止一个工作流编排多个 target kind。
- Settings 可安装 exact Android ADB 目标，包含稳定 slot、设备 serial/identity、能力、consent 与健康状态，不把临时路径或进程句柄写入图。
- Android 安装通过 Slice 13 的平台中立 target Adapter 接入，不为 ADB 复制 Settings、admission、policy、节点选择或 runtime provider 链。
- 3.1 Catalog 的通用自动化节点按真实语义声明 desktop / android target 能力；只在语义不同处提供 Android 专属节点。
- compiler、admission、provider、journal 与 runtime 通过同一目标 slot 闭环到现有 AndroidADBController，不恢复旧 active-target 全局隐式状态。
- 新建工作流可选择 Windows、Android、跨目标或通用模板；模板只预置目录过滤、目标默认值和示例，不创建第二编辑器或第二运行时。
- Android 截图、坐标拾取、模板 BlobRef、点击/拖拽/滚轮/文本、应用启动停止形成可执行闭环；不支持的 key-down/key-up/相对移动明确禁用或诊断。
- ADB 不可用、设备离线、方向/分辨率变化和 stale installation 形成可定位诊断与可操作恢复提示。

## Blocked by

已解除：Slice 13 已完成；产品决定为创建模板，不增加 Workflow Source 平台分类，且允许一个工作流使用多个 target kind。

## Verification

先做 Adapter/controller/provider/Catalog/compiler 的定向集成验证；Slice 完成所在阶段统一运行 task check、task build、ADB emulator smoke、Windows WebView smoke 与人工视觉检查。

## Out of scope

复制旧 Container 编辑器、恢复全局 active target、为 Windows/Android 建两套 Workflow Source 或运行时、真机 iOS、任意 adb shell 节点。

## Result

Completed。Android 通过 Stage 5 已平台中立的 installed Adapter registry 接入 automation.targets、descriptor、provider、policy 与 admission；Profile 固定 ADB serial/product/model/device 和 package，Settings 提供发现、选择、consent 与健康诊断。通用截图、点击、移动、拖拽、滚动、文本、激活节点支持 desktop/android；stop-target-app 仅 android-device，可用性由独立 capability 表达；组合键、相对移动、playback 保持 desktop-only。工作流创建提供通用、Windows、Android、跨目标模板，模板仅过滤目录与显示绑定引导，不分叉 Source/compiler/runtime。资源截图与模板使用 ResolveTarget，录制仍明确限 desktop。task check、task build、Linux/amd64 与 darwin/arm64 cross-compile、Windows WebView smoke 和 bilibili_api35 emulator 真 ADB smoke 全绿。