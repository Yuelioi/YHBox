# Automation targets and controllers

Host OS 与 automation target 是两个维度。Windows/Linux/macOS 是运行 Yotta 的 host；Win32 window、Android ADB、Browser CDP 是被自动化 target。

核心层通过 target descriptor、capability set 和 controller factory 工作。截图、指针、键盘、窗口和应用控制按能力组合；缺失能力返回 typed unsupported/assembly error。Win32 import 被架构测试限制在允许的 adapter package，新增 target 不应迫使 container engine 理解其平台 payload。

Windows 是完整 host。Android adapter 可在受支持 host 上运行。Browser CDP 目前是内部实验能力，不应恢复为普通用户节点，除非权限、生命周期和产品 UX 一并设计。

