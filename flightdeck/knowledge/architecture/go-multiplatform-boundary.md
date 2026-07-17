---
kind: note
summary: "3.1 installed automation 已收敛为 descriptor + target-kind Adapter seam；Win32、Android、Browser 共用注册链，未来 macOS 只新增 Adapter。"
activation: action
read_when: "before adding Linux/macOS/Android/Browser support, moving native code, designing automation targets/controllers/installations, or claiming the Go backend/product is cross-platform."
recheck_when: "automation Settings schema、target taxonomy、installed provider、appbootstrap policy、application identity 或原生宿主 runtime 支持变化后"
---
# Go 多平台边界与自动化安装 seam

Windows 是完整支持平台；Linux/macOS 只承诺平台中立核心可测试、GUI 可编译且为 preview。compile gate 不等于 runtime support，也不能把 unsupported stub 描述成可用 fallback。

## 已成立的 seam

- autostart、admin、console、input、capture、window、hotkey、calibration、recording 与 tools 的 syscall 实现使用 build-tagged Adapter；平台中立 package 不直接 import Win32 binding。
- internal/automation/controller 与 target 定义 Controller、Screenshotter、PointerInput、KeyboardInput、AppLifecycle 和 target/coordinate semantics；Win32、Android、Browser 各自声明真实能力。
- internal/architecture/platform_boundaries_test.go 守住已平台中立的 node/controller/target/execution/expr/llm/script 与工具 package，禁止重新引入 Win32 concrete dependency。

## 已闭合：平台中立 installed automation seam

- Settings durable schema 使用 automation.targets；每项携带稳定 semantic target kind 与独立 Adapter identity。旧 win32Targets 单向迁移并撤销旧 consent。
- Workflow、通用节点、appbootstrap 与 policy 使用 InstallationDescriptor 的 target kind、provider identity、resource kinds 与 operations，不读取 Win32 常量。
- installed 模块的外部 Interface 只暴露 slot、descriptor、sealed profile 与 provider；生产 TargetTypes 与 runtime registry 共用单一 Adapter 注册源，Win32、Android、Browser 与 test Adapter 已证明 seam。
- 标准输入与截图复用 internal/automation/controller；provider 继续集中处理 exact payload、权限、并发、held input、capture budget、journal 与 cleanup。
- Win32 runtime target/HWND、executable/title/class 与 backend 只存在于 Win32 Adapter。Settings profile 显式声明 windows-executable identity；未来 macOS Adapter 可声明 bundle/code-sign identity，而无需修改 Workflow、compiler、scheduler 或 policy。
- Browser Adapter 只接受字面 loopback HTTP discovery origin，固定 exact page id 与同 authority `/devtools/page/{id}` WebSocket；拒绝 redirect、remote/ambient authority 与 identity drift。
- 非 Windows host 通过 per-Adapter availability fail closed，不再把整个 installed module 定义成 Win32-only Interface。

## 新平台接入规则

- 外部安装 Interface 使用稳定 slot、语义 target kind、capability/resource/operation descriptor、sealed typed profile 和 consent；不公开 native handle。
- 内部使用 target-kind registry。Win32、Android、Browser、fake/test 与未来 macOS 是 Adapter；新增能力必须从同一生产注册描述生成 UI descriptor 与 runtime provider。
- 优先让通用节点依赖语义 target kind/capability family，而不是实现名。desktop window 的 Win32/macOS 差异留在 Adapter profile。
- 复用 controller 语义；3.1 resource provider 继续集中处理 exact payload、权限、并发、held input、capture budget、journal 和 cleanup，不能把安全逻辑散回各 Adapter。
- appbootstrap 与 policy 从 Installation descriptor 取 target kind、provider identity 和 resource kinds，不比较全局单一 Win32 常量。
- 通用前端从 backend descriptor 获取已安装 target 和 profile editor；禁止跨视图硬编码 win32Targets。
- Windows HWND、Android serial、Browser target id、macOS CGWindowID/AXUIElement 只存在于 Adapter profile/临时解析结果，不进入 Workflow Source 或 durable journal；Source 只保存 installation slot。
- installed application identity 也必须允许平台 Adapter：Windows executable digest 与 macOS bundle/code-sign identity 不能强塞进同一个 Windows-only表单。

完成平台中立 seam 后，新增 macOS desktop Adapter 不应修改 Workflow Source、通用节点 contract、compiler、scheduler、policy 或运行请求格式。

## 其他平台边界

- installed application lifecycle 只有等价 process/application identity 与 desktop lifecycle Adapter 存在时才安装 provider；其他宿主 fail closed。
- untrusted Script/Process Plugin Host：Windows 必须 LPAC/AppContainer + atomic Job；其他宿主缺少等价 confinement 时不注册 provider。
- root wiring 不直接 import Win32；Wails GUI 仍需各宿主原生工具链。Ubuntu/macOS production compile 必须在原生 runner 验证，首次运行和真实宿主 smoke 是独立门禁。
- 新增跨平台能力必须分别声明 core compile/test、provider installation、GUI compile、真实 runtime smoke 和发布等级。
