# Slice registry

| Slice | 状态 | Blocked by | Outcome |
| --- | --- | --- | --- |
| [01 交互正确性](01-interaction-correctness.md) | completed | 无 | 节点选择、拖拽和位置同步稳定。 |
| [02 连线创作](02-connection-authoring.md) | completed | 01 | 输入输出提示和候选节点创作恢复。 |
| [03 选择与布局](03-selection-layout.md) | completed | 01–02 | 多选、对齐、分布和自动布局恢复。 |
| [04 诊断与 Run trace](04-diagnostics-run-trace.md) | completed | 02 | 编译诊断、运行轨迹和节点定位闭环。 |
| [05 真 Debugger](05-true-debugger.md) | completed | 04 | 断点、暂停、单步和继续闭环。 |
| [06 模板便利节点](06-template-convenience-nodes.md) | completed | 02 | 检测/等待等模板节点适配新架构。 |
| [07 资产创作集成](07-asset-authoring-integration.md) | completed | 06 | 图片、clip 与模板资产可用于节点创作。 |
| [08 Android 目标连续性](08-android-target-continuity.md) | completed | 02 | Android/ADB 工作流模板与目标恢复。 |
| [09 桌面应用目标安装](09-desktop-application-target-installation.md) | completed | 02 | 桌面应用、窗口目标和捕获安装恢复。 |
| [10 工作流库管理](10-workflow-library-management.md) | completed | 无 | 删除、批量、分页等库管理能力补齐。 |
| [11 AI endpoint 安装](11-ai-endpoint-installation.md) | completed | 无 | 自定义 API URL 与模型安装恢复。 |
| [12 Launcher 可发现性](12-launcher-discoverability.md) | completed | 无 | 悬浮窗启动入口恢复。 |
| [13 平台中立自动化安装](13-platform-neutral-automation-installation.md) | completed | 08–09 | desktop/Android/browser 通过 Adapter seam 接入。 |
| [14 基础 UX 回归](14-foundation-ux-regressions.md) | completed | 01–03 | 深色 UI、节点端口、对话框和反馈策略修复。 |
| [15 高级能力决策](15-advanced-capability-decisions.md) | completed | 04–07 | 录制、资源、调试等高级能力纳入 3.1。 |
| [16 Workflow Source 可移植性](16-workflow-source-portability.md) | completed | 13 | 导入导出和 source-native 工作流闭环。 |
| [17 资产库规模](17-asset-library-scale.md) | completed | 07 | 资产搜索、分页与批量管理补齐。 |
| [18 Source-native 多图](18-source-native-multigraph.md) | completed | 16 | 多图、子图调用和导航落地。 |
| [19 Browser CDP 安装](19-browser-cdp-installation.md) | completed | 13 | Browser/CDP 目标安装与执行闭环。 |
| [20 桌面目标 UAC 与授权体验](20-desktop-target-uac-and-consent-ux.md) | blocked | 真实管理员游戏窗口 smoke | 默认管理员、最高权限自启、捕获安装和批量授权。 |
| [21 权威类型系统基础](21-type-system-foundation.md) | completed | 无 | 名义关系、traits、LUB、泛型求解和 Projection parity。 |
| [22 类型与节点能力闭包](22-type-capability-closure.md) | completed | 21 | 结构字段、Break 节点和 Type × Capability 门禁。 |
| [23 Typed Authoring 体验](23-typed-authoring-ux.md) | completed | 22 | 显式转换、typed State、跨图影响预览与 Compiler 门禁。 |
| [24 Settings 引用完整性](24-settings-reference-integrity.md) | completed | 无 | 删除 application 与依赖 target 成为原子操作。 |
| [25 连接计划/Compiler parity](25-connection-plan-compiler-parity.md) | in_progress | 21、23 | 同一固定 fixture 防止 TypeScript/Go 类型边界漂移。 |
