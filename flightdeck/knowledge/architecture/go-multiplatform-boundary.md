---
kind: note
summary: "Yotta automation installation 已由 Adapter 拥有 profile intent/payload、manifest、verifier 与 runtime；新增 target 不改中央 Settings、Policy 或 composition-root capability switch。"
activation: action
read_when: "添加 Linux/macOS/Android/Browser 支持，设计 automation target/controller/installation，修改 Settings automation schema，或声称新增平台只需 Adapter 时"
recheck_when: "Profile intent/payload、Adapter descriptor、Settings fallback editor、installation manifest、Target Runtime generation 或原生宿主支持变化后"
---
# 多平台边界：installation 全链属于 Adapter

Windows 是完整支持 host；Linux/macOS 当前只承诺平台中立核心测试和预览级 GUI compile。Android ADB 与 Browser CDP 是 target，不是 host。compile、stub、controller 单测或 Adapter conformance 都不能提升产品支持等级。

## 已成立的 seam

- Workflow Source 只保存 installation slot，不保存 HWND、PID、ADB handle、CDP session、endpoint、path 或未来 CGWindowID。
- Settings/Profile 使用稳定 envelope：target kind、adapter kind、profile version 与 opaque payload。中央 services 不声明 Win32/ADB/Browser payload struct，也不按平台解码。
- 每个 Adapter registration 拥有 Settings intent codec、sealed runtime payload、profile schema/editor fields、validator/verifier、runtime factory、health 与 capability/resource/operation manifest。
- 同一 sealed installation manifest 派生 authoring descriptor、Admission Host Profile、provider operations、Policy/consent digest 与 health；composition root 不再从 operation/resource kind 反推 capability。
- provider 保存安装时 Adapter 的 verifier。自定义或未来 Adapter 的 authoring、health 和 Invoke 不能回落到 production default registry，否则会在安装成功后被误报 unsupported。
- AutomationTargetRuntime 原子准备和发布 generation；新 Run 只见新代，旧 Run 持 exact lease，空闲旧代回收，Settings 只提交持久化意图。
- Settings 页面为未知 target 提供 descriptor-driven 基础表单；Win32 capture、ADB discovery、Browser page discovery 是可选的专用增强，不是新增 Adapter 的基础阻塞。

## 新 target / host 接入规则

新增 target 实现应只增加 Adapter registration、adapter-owned intent/payload 与必要的 native/controller code。它不得要求修改 Workflow Source、通用 Node Contract、Compiler、scheduler、Policy、中央 Settings schema 或 composition-root capability switch。平台身份留在 Adapter payload：Windows executable、macOS bundle/code-sign identity、Android device/package、Browser endpoint/page 不能被强塞进一个中央 union。

平台支持声明仍需分别证明 core compile/test、installation conformance、GUI compile、真实 runtime smoke 与发布等级。macOS 的 seam 结构已经具备，但在 Slice 36 的 compile fixture 和原生 runner 证据前仍是 architecture proof，不是产品支持。
