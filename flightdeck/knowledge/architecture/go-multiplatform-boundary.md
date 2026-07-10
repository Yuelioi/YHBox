# Go 多平台边界现状
SUMMARY: Yotta 已有 controller capability seam，但 container/runtime、services 和根装配仍直接依赖 Win32；Linux/macOS `go build ./...` 当前失败，多平台工作应先闭合这条 seam。
READ WHEN: before adding Linux/macOS support, moving Win32 code, designing automation targets/controllers, or claiming the Go backend is cross-platform.

---

2026-07-10 源码与交叉编译确认：

- `internal/automation/controller` 已把 Controller、Screenshotter、PointerInput、KeyboardInput、AppLifecycle 拆成平台中立能力 interface，并已有 Win32、Android ADB、Browser CDP adapter。这是应保留的方向。
- `internal/services/container/runtime.RuntimeContext` 仍直接持有 `pkg/input.Backend`、`pkg/capture.IBackend`、`winutil.WindowHandle`，并 import `github.com/lxn/win`。
- 只有少数 inputclip/QPC/ADB 文件具备 build tag，平台依赖没有形成 package 级隔离。
- 初始 Linux/Darwin build 同时被 `lxn/win` 与 Windows registry 阻断；registry 已通过 autostart 平台隔离消除，当前首个项目内阻断是剩余 `lxn/win` 依赖。

已完成的收敛：

- autostart、admin、console、shell 已拆成 Windows 实现与非 Windows 实现，`pkg/platform` 可在 Windows/Linux/macOS 编译。
- `internal/architecture/platform_boundaries_test.go` 守住已平台中立的 node/controller/target/execution/expr/llm/script 与纯工具 package，禁止重新 import Win32/input/capture/winutil。
- 完整 Linux 构建仍有 hotkey、input、capture、winutil、container/runtime、recording/tools 等旧 Win32 链；Wails alpha.91 的 GUI package 在 `CGO_ENABLED=0` 下本身也无法构建，宿主 GUI 与 backend portability 必须分开验收。

后续多平台工作的验收应是：domain/node/container runtime 不再 import Win32 包，Windows 能力由 adapter 提供，且 Windows/Linux/macOS CI 至少都能 `go build ./...`。仅增加 build tag 或 stub、但仍让 runtime 同时理解 controller 与旧 Win32 backend，不算 seam 闭合。
