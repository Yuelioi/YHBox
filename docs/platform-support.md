# Platform support

| Host / target | Status | CI guarantee | Notes |
|---|---|---|---|
| Windows 11 x64 host | Supported | Full Go tests, vet, staticcheck, coverage, race-sensitive packages | Win32 window, background input/capture and WGC packaging |
| Linux x64 host | Preview | Platform-neutral tests pass locally; native GUI job configured, first remote validation pending | Windows-only capabilities return typed unsupported errors |
| macOS arm64 host | Preview | Platform-neutral tests pass locally; native GUI job configured, first remote validation pending | Packaging/signing and automation permissions are not productized |
| Android ADB target | Supported adapter on a supported host | Contract and service tests | Android is an automation target, not a host OS |
| Browser CDP target | Internal/experimental | Package tests | Not exposed as a normal user target node |

“CI 可编译”不等于完整产品支持。提升平台等级需要安装、权限、输入/截图能力、升级和真实设备集成测试全部闭合。
