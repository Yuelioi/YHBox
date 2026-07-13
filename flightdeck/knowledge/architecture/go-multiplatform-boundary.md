---
kind: note
summary: "Yotta backend core 已闭合 Win32/Wails seam 并进入 Windows/Linux/macOS 门禁；完整 GUI 必须按宿主 OS 使用 Wails 原生依赖单独验收。"
activation: action
read_when: "before adding Linux/macOS support, moving Win32 code, designing automation targets/controllers, or claiming the Go backend is cross-platform."
---
# Go 多平台边界现状
2026-07-11 源码、依赖图与交叉编译确认：

- `internal/automation/controller` 已把 Controller、Screenshotter、PointerInput、KeyboardInput、AppLifecycle 拆成平台中立能力 interface，并已有 Win32、Android ADB、Browser CDP adapter。这是应保留的方向。
- `internal/services/container/runtime.RuntimeContext` 已只经 controller/provider capability 使用 input/capture；窗口元数据归属 `automation/target`，runtime core 不再理解 legacy backend bootstrap。
- 初始状态只有少数 inputclip/QPC/ADB 文件具备 build tag，平台依赖没有形成 package 级隔离。
- 初始 Linux/Darwin build 同时被 `lxn/win` 与 Windows registry 阻断；registry 已通过 autostart 平台隔离消除，当前首个项目内阻断是剩余 `lxn/win` 依赖。

已完成的收敛：

- autostart、admin、console、shell 已拆成 Windows 实现与非 Windows 实现，`pkg/platform` 可在 Windows/Linux/macOS 编译。
- `pkg/input` 已把公共 Backend contract、Windows adapter factory 与非 Windows unsupported factory 分离；Linux/Darwin 依赖图不再包含 `lxn/win`。
- `pkg/capture` 已把公共 IBackend contract、GDI/WGC Windows adapter 与非 Windows factory 分离；mock capture 保持跨平台可用，Linux/Darwin 依赖图不再包含 `lxn/win`。
- 非 Windows autostart/shell/input/capture 使用可被 `errors.Is(err, platform.ErrUnsupported)` 分类的 typed error，不再用 nil/panic 或不可识别字符串表达不支持。
- 纯 `VK` 键名解析已从 Win32 syscall 文件移入跨平台源码；非 Windows `KillProcess` 返回 typed unsupported，因此 `internal/nodes/input` 与 `internal/nodes/io` 已可在 Linux/Darwin 编译测试。
- `WindowHandle` / `WindowMatchSpec` 已移入平台中立 target contract；`pkg/winutil` 已拆成公共匹配逻辑、`_windows.go` 实现与非 Windows typed unsupported。
- container/runtime 的 Win32 window 校验、解析、控制与 borderless state 已封装在 `_windows.go` adapter；core 与 tests 不再直接依赖 `lxn/win`，Linux/Darwin 可编译 container/runtime tests。
- `internal/architecture/platform_boundaries_test.go` 守住已平台中立的 node/controller/target/execution/expr/llm/script 与纯工具 package，禁止重新 import Win32/input/capture/winutil。
- hotkey、calibration、recording、tools 都已有平台 adapter；`internal/services` 不再 import Wails，并由架构测试守卫。Linux/macOS portable-core CI 已覆盖这些 backend package。
- root wiring 不再直接 import `lxn/win`；完整 root package 在 `CGO_ENABLED=0` 下的剩余失败来自 Wails GUI runtime。它不是 portable backend 回退：官方 Linux GUI 基线要求 gcc、GTK4、WebKitGTK 6.0（旧栈可用 `gtk3` tag），macOS 要 Xcode command line tools。
- Wails library、release CLI 与 README 安装命令统一固定为 `v3.0.0-alpha2.117`，`scripts/verify-wails-version.ps1` 防止三处再次漂移。

后续多平台工作的验收分两层：portable backend 继续保持三平台测试与禁依赖守卫；GUI 已在 Ubuntu 24.04 amd64 与 macOS 15 arm64 原生 runner 安装宿主依赖、生成前后端产物并以 production tag 编译。compile gate 不等于运行时支持声明；首次远端运行与对应宿主 smoke 仍必须完成。不要用 `CGO_ENABLED=0 go build ./...` 代替 GUI 验收。
