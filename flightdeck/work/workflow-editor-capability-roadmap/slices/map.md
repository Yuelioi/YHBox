# Slice registry

| Slice | 状态 | Blocked by | Outcome |
| --- | --- | --- | --- |
| [01 交互正确性](../01-interaction-correctness.md) | completed | 无 | 点击、连线和控件交互不再导致节点跑偏，真实拖拽只提交一次。 |
| [02 类型感知连线](../02-connection-authoring.md) | completed | Slice 1 | 共用权威规则完成 hover、拖空白候选和原子创建连线。 |
| [03 选择与布局](../03-selection-layout.md) | completed | Slice 1 | 恢复多选、批量操作、对齐分布、吸附与 LR/TB 布局。 |
| [04 诊断与运行轨迹](../04-diagnostics-run-trace.md) | completed | Stage 1 | compiler diagnostics 与 journal 状态可定位到节点。 |
| [05 真调试器](../05-true-debugger.md) | completed | Slice 4 | 在唯一 scheduler 上提供断点、暂停、单步、继续和停止。 |
| [06 模板复合节点](../06-template-convenience-nodes.md) | completed | Stage 2 | 用 exact target 与 BlobRef 恢复等待/点击模板节点。 |
| [07 资源预览与录制](../07-asset-authoring-integration.md) | completed | Slice 6 | 受限预览、资源失效诊断和录制草稿接入 3.1 创作闭环。 |
| [14 基础 UI 与交互回归](../14-foundation-ux-regressions.md) | completed | Stage 3 | 关闭暗色、端口、alert/toast、Start/Delete/状态等回归。 |
| [09 桌面应用与窗口目标安装](../09-desktop-application-target-installation.md) | completed | Stage 3 | 修复取消/空白默认并接回 F9 exact window capture。 |
| [10 工作流库管理](../10-workflow-library-management.md) | completed | Stage 3 | 提供删除、批量、搜索、排序与分页的工作流管理。 |
| [11 AI endpoint 安装](../11-ai-endpoint-installation.md) | completed | Stage 3 | 在可信 AI installation 中配置 endpoint。 |
| [12 悬浮窗入口](../12-launcher-discoverability.md) | completed | Stage 3 | 接回主界面可发现入口和既有 launcher window。 |
| [13 平台中立 automation seam](../13-platform-neutral-automation-installation.md) | completed | Stage 4 | 以多 Adapter 深模块替换 Win32 专用安装 Interface。 |
| [08 Android 目标连续性](../08-android-target-continuity.md) | completed | Slice 13 | 恢复 Android exact installation、创作与 runtime。 |
| [15 高级能力恢复](../15-advanced-capability-decisions.md) | in_progress | Stage 6 | 直到 Slices 16–19 完成才关闭 3.1 高级能力缺口。 |
| [16 Workflow Source 导入导出](../16-workflow-source-portability.md) | completed | commit aaa34711 | canonical bundle、完整性验证、身份冲突与产品入口。 |
| [17 资产规模化与安全清理](../17-asset-library-scale.md) | completed | Slice 16 | 分页、批量、variant 维护与完整 Blob root GC。 |
| [19 Browser CDP 产品闭环](../19-browser-cdp-installation.md) | in_progress | Stage 5 | exact installation、Settings、provider/Catalog 与真机 smoke。 |
| [18 多图创作](../18-source-native-multigraph.md) | pending | Slice 16 | subgraph 调用/创作、comment 与 presentation reroute。 |
