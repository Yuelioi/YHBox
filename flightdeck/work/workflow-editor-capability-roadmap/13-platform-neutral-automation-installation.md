---
slice: "13"
title: 平台中立的自动化安装 seam
status: completed
---

# Slice 13：平台中立的自动化安装 seam

## Outcome / Question

把 3.1 installed automation 从 Win32 专用模块深化为平台中立的安装模块：Workflow、节点、admission 和通用 UI 只理解语义能力与稳定 target slot，Win32、Android、Browser 和未来 macOS 通过真实的 Adapter seam 接入。

## Completion criterion

- 记录并消除平台泄漏：automation.win32Targets、Win32-only ProfileDraft、TargetKind 常量、ResolveWindow driver Interface、platform_other 全禁用、节点只允许 win32-window、policy 比较单一 TargetKind、前端硬读 win32Targets。
- 决定并迁移 target taxonomy：优先采用稳定语义 kind（如 desktop-window）与独立 adapter/platform identity；若保留平台 kind，必须用 capability family 避免每加 OS 修改所有通用节点。
- Settings 改为统一 automation.targets 集合和带 kind 的判别式 profile；保留严格 typed schema、canonical digest、预算、consent invalidation，并提供 win32Targets 数据迁移。
- installed 模块提供小而深的安装 Interface；内部 target-kind registry 至少有 Win32 与测试/Android 两个 Adapter，证明 seam 真实存在。
- 复用 internal/automation/controller 的 Controller、Screenshotter、PointerInput、KeyboardInput 与 AppLifecycle 语义，不再维护一条平行且 Win32 专用的运行模型；3.1 provider 继续负责权限、并发、payload 校验、held input 和 journal 边界。
- appbootstrap 与 policy 从每个 Installation 的 descriptor 读取 target kind、resource kinds、operations、provider identity，不引用单一 automationinstalled.TargetKind。
- 节点 capability 面向语义 operation/resource；增加 macOS desktop Adapter 时不修改 Workflow Source、通用节点 contract、compiler、scheduler、policy 或运行请求格式。
- 前端从 backend installation descriptors 获取 kind、label、能力和 profile editor，不在 WorkflowInspector、AssetsView、Recording 或 Settings store 中硬编码 win32Targets。
- 平台原生 identity 留在 Adapter：Windows 使用 executable/HWND/title/class，macOS 可使用 bundle ID、code-sign identity、CGWindowID/AXUIElement；任何 native handle 都不进入图或 durable journal。
- 同时确认 installed application lifecycle 的 executable-only profile 是否足以承载 macOS .app/bundle identity；不足则建立相邻 Adapter seam，不让窗口目标反向依赖 Windows executable 模型。

## Blocked by

Stage 3 批量验收；target kind 采用语义 kind 还是 capability family 的架构决策。

## Verification

先用 Win32 Adapter 与第二个 fake/Android Adapter 做 installation/profile/provider/admission/authoring conformance；随后运行平台边界测试、Linux/macOS core compile、Windows 定向 runtime test。所属阶段末统一运行 task check、task build、Windows smoke；macOS 只在原生 runner smoke 后声明 runtime 支持。

## Out of scope

本 Slice 不实现完整 macOS 输入/捕获，不恢复 ambient active-window，不把 profile 退化成未校验 map，不改变 Workflow Source 为平台专属格式，不复制 provider/runtime。

## Result

Completed。Settings 统一为带 semantic targetKind 与 adapterKind 的 automation.targets，并从 win32Targets 单向迁移且撤销旧 consent。installed 模块提供 descriptor + Adapter registry 的深 Interface；Win32 与 test Adapter 通过同一安装 conformance，driver Interface 改为 ResolveTarget，标准输入/截图复用 automation/controller。appbootstrap、policy、节点和前端改为读取 descriptor/semantic kind；Windows executable identity 被显式声明为 Adapter 接受的 identity kind，未来 macOS 可新增 desktop-window Adapter 与 bundle identity 而不修改 Workflow/compiler/scheduler/policy。Linux/amd64 与 darwin/arm64 core 交叉编译、task check、task build、Windows WebView smoke 与截图目检均通过。
