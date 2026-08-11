# Platform support

| Host / target | Product status | Implementation and verification boundary |
| --- | --- | --- |
| Windows x64 host | Supported production path | 仓库支持策略是 Windows 11 x64；代码和 package 只证明 Windows amd64 production path，并不在运行时强制检查 OS 版本。CI 运行 `task check:full`、Windows race 和 production package；最终发布仍需 native automation/plugin/storage/WebView smoke |
| Linux x64 host | Preview | Ubuntu CI 运行选定 portable core，并 production-tag 编译 GUI；无产品 package、签名、secure store/process sandbox 或 native GUI smoke |
| macOS arm64 host | Preview | macOS CI 运行选定 portable core，并 production-tag 编译 GUI；无产品 package、签名、secure store/process sandbox、权限 UX 或 native GUI smoke |
| `desktop-window` / `win32` | Windows implementation | descriptor 的 `HostAvailable` 只在 Windows 为 true；非 Windows 返回 typed unsupported，不存在替代 native adapter |
| `android-device` / `android-adb` | Implemented, external smoke required | registry/profile/controller/driver 与纵向测试存在；发布结论需要已授权 exact serial/package 真机或模拟器 smoke |
| `browser-cdp` / `browser-cdp` | Implemented, external smoke required | registry/profile/controller/driver 与纵向测试存在；发布结论需要隔离 Chrome/Edge profile 的真实 smoke |

表中的结论来自当前 build tags、`.github/workflows/ci.yml`、Task 和 `internal/automation/installed` 生产注册表，
不根据旧路线图推断。Windows production manifest 请求 `requireAdministrator`；开发 manifest 使用
`asInvoker`，两者也不能互相代表。
“能编译”、adapter 已注册或 UI 有入口都不等于完整支持。提升等级需要安装、权限、配置创作、输入/捕获、
生命周期、失败恢复和真实 host/target 黄金旅程全部闭合。对应命令与触发条件见
[构建与验证指南](../flightdeck/knowledge/build/build.md)。
