# Yotta 3.1 产品升级执行计划

## 完成定义

“3.1 major upgrade 完成”必须同时满足：

- 旧产品能力已按入口、管理、创作、运行四层完成机械对比。
- 所有 P0/P1 能力明确为已恢复、由新能力替代或经产品决策放弃，不能处于“后端还在但入口断了”状态。
- Windows 完整支持链通过正式 build/WebView/关键真机 smoke。
- Android 恢复完整安装、创作和运行闭环。
- automation installation 已平台中立，未来 macOS 只需新增 Adapter，不需要修改 Workflow、通用节点、compiler、scheduler、policy 或通用前端。
- 全量门禁按阶段批量运行并留下可信结果。

## 执行原则

- 一个 Stage 包含多个相邻 Slice；Slice 只作为实现、恢复和提交边界。
- Slice 内仅运行支撑继续开发的定向测试、typecheck 或 compile。
- Stage 完成后统一执行阶段聚合测试、task check、task build 和触发的真实宿主 smoke。
- 不因每个小按钮重复跑全量验收。
- 保存等原地动作成功不 toast；失败才使用统一 Nuxt UI。
- 不恢复旧 Container runtime，不创建第二套 Workflow Source 或执行器。

## Stage 0：能力审计与架构纠偏（完成）

交付：

- 以 9fce7870^ 为旧产品基线的机械能力对比。
- 3.1 产品连续性四层审计规则。
- Win32 平台泄漏分析和平台中立 automation seam 方案。
- Slices 1–14 注册表及依赖关系。

验收：Flightdeck 文档完整、每个批准 Slice 有 Outcome、Completion、Blocked by、Verification、Out of scope 和 Result。

## Stage 1：可靠而高效的图编辑（完成）

Slices 1–3：

- 交互位置正确性。
- 类型感知连线与拖空白候选。
- 多选、批量操作、clipboard、对齐、分布、吸附、LR/TB 自动布局。

已完成 task check、build、Windows WebView smoke。

## Stage 2：运行认知与真正调试（完成）

Slices 4–5：

- 结构化 compiler diagnostics 与 journal timeline。
- 同一 scheduler 上的断点、暂停、单步、继续和停止。

已完成 task check、build、Windows WebView smoke。

## Stage 3：自动化创作闭环（完成）

Slices 6–7：

- WaitTemplate、WaitTemplateGone、ClickTemplate。
- 安全 Blob 缩略图、资源失效诊断。
- 简易按键/点击录制、轨迹录制草稿。
- 预览后批量插入，一个 undo，不自动保存。

验收结果：相关 Go/前端聚合测试、task check、最终 task build 与 Windows WebView smoke 全部通过；编辑器和资源页截图已人工目检。隔离 smoke 无已安装 Win32 target，真实 OS 输入捕获按 native-target smoke 触发条件执行。

## Stage 4：基础产品可用性与桌面连续性（完成）

顺序：

1. Slice 14：暗色、端口、alert/toast、Start/Delete、目录搜索、Run 状态和位置回归。
2. Slice 9：桌面应用选择取消、新目标空白默认、F9 捕获窗口。
3. Slice 10：工作流删除、批量、搜索、排序、分页与引用语义。
4. Slice 11：可信 AI installation 的自定义 endpoint。
5. Slice 12：主界面和设置页恢复悬浮窗入口。

验收结果：Stage 4 全部 Slices 已完成；task check、正式 task build 与隔离 Windows WebView smoke 均通过。烟测实际创建工作流、点击/拖放节点并检查 103 个目录节点、3 个画布节点、AI review、资源/录制入口；编辑器和资源页 PNG 已人工目检。文件选择取消、F9 捕获、launcher 重复打开聚焦由宿主边界测试覆盖，烟测未读取或删除用户 data。

阶段门禁：相关前端/Settings/workflow service 聚合测试 → task check → task build → 正式 Windows 文件对话框/F9/WebView/launcher smoke → 人工视觉检查。

## Stage 5：平台中立 automation architecture（完成）

Slice 13：

- automation.targets 判别式安装集合。
- target-kind Adapter registry。
- 复用 controller 语义，保留 provider 权限/并发/审计深度。
- appbootstrap/policy/前端由 installation descriptor 驱动。
- 迁移 win32Targets，证明 Win32 + fake/Android 两个 Adapter conformance。
- 审计 installed application identity 对 macOS bundle/code-sign identity 的扩展点。

验收结果：Win32 与 test Adapter 通过同一 installation conformance；旧 settings 迁移、consent invalidation、descriptor-driven policy/bootstrap/frontend、controller 复用与平台边界测试通过；Linux/amd64、darwin/arm64 core test binary 成功交叉编译。task check、正式 task build 与 Windows WebView smoke 全部通过，截图已人工目检。

## Stage 6：Android/ADB 产品闭环（完成）

Slice 8：

- exact ADB installation、设备健康与 consent。
- Settings 与工作流模板入口。
- 通用节点 Catalog/compiler/provider/runtime 接入。
- 截图、坐标、模板、点击/拖拽/滚轮/文本、应用启动停止。
- emulator 离线、旋转、分辨率变化和 stale installation 诊断。

验收结果：exact ADB identity/package、Settings 设备发现与健康、统一 Adapter/provider、通用/Android-only capability、创建模板引导和资产截图已接通；不支持的组合键、相对移动与低级回放在 Catalog/admission 层禁用。task check（144 前端测试、65.4% 全局 Go coverage）、正式 task build、Linux/amd64 与 darwin/arm64 installed core cross-compile、Windows WebView smoke（104 catalog nodes）及 bilibili_api35 emulator 的启动/停止、PNG 截图、移动、点击、拖拽、滚动、文本 smoke 全绿。

## Stage 7：高级能力决策与恢复（完成）

完成项：

- 恢复 Source-native 画布节点定位：工具栏、Ctrl/⌘+F、跨 graph 搜索、选中与居中。
- 明确不恢复旧 Container zip、旧 runtime、任意宿主 JS/yt console 和语义 reroute node。
- canonical Source portability、资产规模化管理、完整 subgraph authoring、Browser CDP installation 分别形成 post-3.1 产品边界，不以隐藏后端冒充已支持。
- 旧版实际只有 ExportPackage，没有 ImportPackage 闭环；旧 Asset Maintenance 实际清理 subgraph，决策已按代码事实修正。

验收结果：task check 全绿（145 前端测试、1595 i18n keys、65.4% Go 总覆盖）；task build 生成正式 bin/Yotta.exe；Windows WebView smoke 通过并人工检查编辑器/资源页截图，104 个目录节点、3 个画布节点和“定位节点”入口可见。

## 发布阈值

Stage 1–7 已全部完成并通过各自阶段门禁，可以称 3.1 major upgrade 的既定迁移计划完成。这里不等于“恢复旧版所有功能”：release notes 必须明确列出当前没有 Workflow import/export、资产批量分页、完整 subgraph authoring 和 Browser CDP installation；旧 Container runtime、任意宿主脚本 console 与语义 reroute node 明确不恢复。

未来恢复项必须继续遵守唯一 Workflow Source/compiler/runtime、exact installation/capability、immutable BlobRef 与阶段末批量验收原则。
