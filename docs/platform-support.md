# Platform support

| Host / target | Product status | Verification boundary |
| --- | --- | --- |
| Windows 11 x64 host | Supported | 完整质量门禁、Windows race、production GUI/build/package 与 native automation/plugin/storage/WebView smoke |
| Linux x64 host | Preview | CI 运行平台中立 core 并编译 native GUI；Windows-only capability 必须返回 typed unsupported |
| macOS arm64 host | Preview | CI 运行平台中立 core 并编译 native GUI；签名、权限和 automation 体验尚未产品化 |
| Android ADB target | Release candidate | contract/integration tests；发布结论还需要已授权真机/模拟器纵向 smoke |
| Browser CDP target | Product-integrated preview | contract/integration tests；发布结论还需要 Chrome/Edge 独立 profile 纵向 smoke |

“能编译”、adapter 已注册或 UI 有入口都不等于完整支持。提升等级需要安装、权限、配置创作、输入/捕获、
生命周期、失败恢复和真实 host/target 黄金旅程全部闭合。对应命令与触发条件见
[构建与验证指南](../flightdeck/knowledge/build/build.md)。
