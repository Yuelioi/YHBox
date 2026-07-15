---
kind: note
summary: "Yotta backend core 已闭合 Win32/Wails seam 并进入 Windows/Linux/macOS 门禁；完整 GUI 必须按宿主 OS 使用 Wails 原生依赖单独验收。"
activation: action
read_when: "before adding Linux/macOS support, moving Win32 code, designing automation targets/controllers, or claiming the Go backend is cross-platform."
---
# Go 多平台边界与发布声明

Windows 是完整支持平台；Linux/macOS 只承诺平台中立核心可测试、GUI 可编译且为 preview。compile gate 不等于 runtime support，也不能把 unsupported stub 描述成可用 fallback。

- autostart、admin、console 和通用 platform seam 使用 Windows implementation 与 non-Windows typed unsupported implementation；平台中立 package 不直接 import Win32 binding。
- input、capture、window、hotkey、calibration、recording 与 tools 通过公共 contract/factory 接入；Win32 syscall 留在 build-tagged adapter。
- controller/target contract 位于 internal/automation，Android、Browser 与 Win32 能力由各自 adapter 显式声明；NeedsWindow 不能冒充通用 target requirement。
- internal/architecture/platform_boundaries_test.go 守住已平台中立的 node/controller/target/execution/expr/llm/script 与工具 package，禁止重新引入 Win32/input/capture/winutil concrete dependency。
- installed application lifecycle 只在 Windows 安装 provider；Linux/macOS 保留 authoring contract，但在等价 process identity 与 desktop lifecycle 实现存在前 fail closed。旧 RunProgram、KillProcess、ShellExecuteW 与 taskkill 平台 API 已删除，不得用 typed unsupported stub 复活。
- untrusted Script/Process Plugin Host 是另一条边界：Windows 必须 LPAC/AppContainer + atomic Job；其他宿主缺少等价 confinement 时不注册 provider，不能降级到普通 subprocess。
- root wiring 不直接 import Win32；Wails GUI 仍需要各宿主的原生工具链与库。Ubuntu/macOS production compile 必须在原生 runner 验证，首次运行和真实宿主 smoke 仍是独立门禁。
- Wails library、release CLI 与 README 安装命令统一固定版本，版本同步脚本防止漂移。

新增跨平台能力时必须分别声明：core compile/test 状态、provider 是否安装、GUI compile 状态、真实 runtime smoke 状态和发布等级。不得用 CGO_ENABLED=0 go build ./... 代替 GUI 验收。
