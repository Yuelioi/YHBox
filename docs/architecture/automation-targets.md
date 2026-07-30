# Automation targets and controllers

Host OS 与 automation target 是两个维度。Windows/Linux/macOS 是运行 Yotta 的 host；Win32 window、Android ADB、Browser CDP 是被自动化 target。

核心层通过 target descriptor、resource/operation descriptor 和 controller factory 工作。截图、指针、键盘、窗口和应用控制按 operation 组合；缺失 operation 返回 typed unsupported/assembly error。Win32 import 被架构测试限制在允许的 adapter package，Workflow Source、Compiler 和 scheduler 不理解原生 payload。

Win32、Android 与 Browser 通过同一 adapter registry 配置。每个 Adapter 拥有版本化 profile intent/payload codec、schema/sealer、authoring/editor descriptor、resource manifest、runtime factory、health/availability 和 cleanup；中央 Settings 只持久化 slot、target kind、adapter kind、profile version 与 opaque payload。未知 target 在设置页由 manifest fields 提供基础表单，平台专用捕获/发现组件只是可选增强。

同一配置快照派生 authoring targets、provider registry 与 health。Automation Target 不投影进 Admission Host Profile、Policy、Consent 或 Run Grant，也没有 identity pinning 或过期时间。Node Contract 只声明所需 slot、resource kind 与 operations；执行器在每次 Run 中通过 `internal/targetruntime` 直接打开、调用并关闭配置的 provider。Workflow Source、Compiler、scheduler 和通用 Settings schema 都不需要理解 Win32、ADB、CDP 或未来 macOS payload。

设置提交通过 generation runtime 原子发布；正在运行的 Run 持有旧配置代的生命周期引用，新 Run 只见新代，空闲旧代回收。这个引用只保证热替换时对象仍可用，不是授权、租约或超时。新增和修改配置不要求重启应用。

Android serial/package、Browser discovery endpoint/target ID 和 Win32 selector 都是用户配置。运行时不把设备 product/model、浏览器 WebSocket 地址或可执行文件摘要当作固定身份；配置当前解析到什么对象，adapter 就操作什么对象。Browser CDP 接受本机或远程 HTTP/HTTPS discovery endpoint 以及 `ws`/`wss` 地址。

Windows 是完整 host。Android adapter 可在受支持 host 上运行，但当前发布支持仍取决于 ADB emulator/device 黄金旅程。Browser CDP 的发布等级同样取决于配置、生命周期和真实 Chrome/Edge smoke，而不是 controller/package tests。
