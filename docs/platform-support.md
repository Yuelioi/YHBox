# Platform support

| Host / target | Status | CI guarantee | Notes |
|---|---|---|---|
| Windows 11 x64 host | Supported | Full Go tests, vet, staticcheck, coverage, race-sensitive packages | Win32 window, background input/capture and WGC packaging |
| Linux x64 host | Preview | Platform-neutral tests pass locally; native GUI job configured, first remote validation pending | Windows-only capabilities return typed unsupported errors |
| macOS arm64 host | Preview | Platform-neutral tests pass locally; native GUI job configured, first remote validation pending | Packaging/signing and automation permissions are not productized |
| Android ADB target | Release candidate, not yet release-validated | Contract/service tests; emulator/device journey pending | Android is an automation target, not a host OS |
| Browser CDP target | Product-integrated preview | Package/integration tests; Chrome/Edge host smoke pending | Installation and UI exist, release support is not yet proven |

“CI 可编译”、adapter 注册或页面入口不等于完整产品支持。提升平台等级需要安装、权限、创作绑定、输入/截图能力、生命周期和真实设备/浏览器黄金旅程全部闭合。当前 3.1 架构恢复期间，历史 roadmap 的 completed 状态不能单独提升本表等级。
