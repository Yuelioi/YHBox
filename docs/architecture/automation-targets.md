# Automation targets and controllers

Host OS 与 automation target 是两个维度。Windows/Linux/macOS 是运行 Yotta 的 host；Win32 window、Android ADB、Browser CDP 是被自动化 target。

核心层通过 target descriptor、capability set 和 controller factory 工作。截图、指针、键盘、窗口和应用控制按能力组合；缺失能力返回 typed unsupported/assembly error。Win32 import 被架构测试限制在允许的 adapter package，Workflow Source、Compiler 和 scheduler 不理解原生 payload。

Win32、Android 与 Browser 通过同一 adapter registry 安装。每个 Adapter 拥有版本化 profile intent/payload codec、schema/sealer、authoring/editor descriptor、capability manifest、runtime factory、health/availability 和 cleanup；中央 Settings 只持久化 slot、target kind、adapter kind、profile version 与 opaque payload。未知 target 在设置页由 manifest fields 提供基础表单，平台专用捕获/发现组件只是可选增强。

同一 sealed installation manifest 派生 authoring targets、Admission Host Profile、provider registry、Policy/consent digest 与 health。composition root 不再手工把 operations/resource kinds 翻译成第二套 capability inventory；Workflow Source、Compiler、scheduler 和通用 Settings schema 都不需要理解 Win32、ADB、CDP 或未来 macOS payload。

设置提交通过 generation runtime 原子发布；正在运行的 Run 持有旧 generation lease，新 Run 只见新代，空闲旧代回收。正常安装、修改和授权不要求重启应用。

Windows 是完整 host。Android adapter 可在受支持 host 上运行，但 当前发布支持仍取决于 ADB emulator/device 黄金旅程。Browser CDP 已有产品接入代码，发布等级同样取决于 installation、权限、生命周期和真实 Chrome/Edge smoke，而不是 controller/package tests。
