---
slice: "36"
title: Android、Browser 与平台 Adapter 连续性
status: pending
---

# Slice 36：Android、Browser 与平台 Adapter 连续性

## Outcome / Question

恢复 Android/ADB 与 Browser CDP 的产品闭环，并用最小 macOS descriptor/compile proof 证明新增平台不需要修改 workflow core 或中央 ProfileDraft。

## Completion criterion

- Android adapter 自己定义 profile schema/version、seal、health、editor descriptor 和 runtime factory。
- ADB input/capture/template/clip 使用统一 manifest/admission/journal 路径。
- Browser exact endpoint/page installation 与 operation narrowing 可运行并可诊断。
- 最小 macOS no-runtime adapter 可以注册/编译；新增它不修改 Source、通用节点、compiler、scheduler 或 central profile union。

## Blocked by

Slices 31、34–35。

## Verification

- G16 Android emulator/device matrix、G17 Chrome/Edge controlled smoke。
- Windows golden journeys 回归不变。
- cross-platform core tests、GUI compile 和 adapter conformance 在 Stage R4 末批量执行并形成 commit。

## Out of scope

- 不承诺本次实现 macOS native automation runtime。
- 不把仅有 controller/client 或 WebView 可见当作平台支持。
- 不允许 unsupported platform 静默 fallback 到 desktop Windows profile。

## Result

Pending。
