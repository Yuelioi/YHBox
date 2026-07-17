---
topic: workflow-editor-capability-roadmap
title: 3.1 产品能力连续性审计与升级路线
summary: 审计旧产品能力与 3.1 现状，按唯一新架构、用户体验和必要性决定恢复、重做、延期或删除，并分阶段补齐可见入口、管理流程、创作绑定与运行闭环。
---

## State

3.1 已完成 Workflow Source、compiler/scheduler、exact installation/capability、运行日志与调试等底层架构升级，但产品迁移没有完成。现已用删除旧产品栈的提交 9fce7870 及其父提交做机械化新旧能力基线，并确认当前 3.1 installed automation 的产品与运行 Interface 是 Win32 专用。正式阶段顺序、发布阈值和批量验收门禁见 [upgrade-plan.md](upgrade-plan.md)。

Stage 1–7 已全部完成并通过阶段批量门禁。Stage 7 恢复了 Source-native 画布节点定位，并把剩余高级能力逐项收口为现有替代、post-3.1 工作包或明确不恢复；3.1 不再用隐藏 controller/schema 字段冒充产品能力。

完整结论见 [capability-audit.md](capability-audit.md)，新旧能力逐项表见 [artifacts/legacy-product-capability-diff.md](artifacts/legacy-product-capability-diff.md)，完整 Slice 注册表见 [slices/map.md](slices/map.md)。

## Next

既定 3.1 major upgrade 迁移计划已完成。发布时按 Slice 15 的边界写 release notes；后续若启动 Source portability、资产规模化、subgraph authoring、Browser CDP 或 macOS Adapter，应建立新的独立 Topic，不重新打开本 Topic。

## Read now

- work/workflow-editor-capability-roadmap/upgrade-plan.md

## Read if

- work/workflow-editor-capability-roadmap/upgrade-plan.md — 调整阶段、发布阈值或批量验收门禁
- work/workflow-editor-capability-roadmap/artifacts/legacy-product-capability-diff.md — 判断旧能力是否保留、缺失、替代或明确放弃
- work/workflow-editor-capability-roadmap/slices/map.md — 调整路线优先级、选择后续 Slice 或改变 blocker
- work/workflow-editor-capability-roadmap/capability-audit.md — 判断旧能力去留或新增产品回归
- knowledge/architecture/go-multiplatform-boundary.md — 修改 automation installation、平台 target、应用身份或跨平台声明
- knowledge/architecture/installed-input-authority.md — 实现 Android、Win32 或 macOS 输入目标
- knowledge/architecture/provider-native-ai-installations.md — 实现 AI 自定义 endpoint
- knowledge/build/code-style.md — 开始修改或验证产品代码
- knowledge/frontend/ui.md — 修改产品入口、列表、设置或编辑器 UI

## Progress

- 3.1 的有效升级集中在唯一 Workflow Source/Program 执行路径、内容寻址契约、exact target/capability admission、durable run journal 与同调度器调试；这些底层成果保留。
- Stage 1 已恢复交互正确性、类型感知连线、多选/剪贴板/对齐/分布/自动布局；Stage 2 已交付结构化诊断、运行时间线与真正调试器，两个阶段均通过 task check、build 与 Windows WebView smoke。
- Stage 3 已恢复 WaitTemplate、WaitTemplateGone、ClickTemplate、安全 Blob preview、失效资源诊断、按键/点击与轨迹录制草稿、预览后一次 undo 批量插入。task check、最终 task build 与 Windows WebView smoke 已通过。
- Stage 4 Slice 14 已关闭基础回归：暗色/端口/Start/目录/Run/位置问题经现有探针确认，新增 edge 键盘删除、target 安装直达、节点说明与工作流行内 Run 反馈；定向 9 tests、i18n 与 typecheck 通过。
- Stage 4 Slice 9 已恢复桌面安装连续性：文件选择取消是无副作用 no-op，新窗口目标从空白草稿开始，复制为显式动作；设置页接回可取消/超时的 F9 捕获，并按可执行路径与 SHA-256 精确匹配已安装应用后回填 title/class。tools/winutil、相关前端测试、1517-key i18n 与 typecheck 通过。
- Stage 4 Slice 10 已把首页升级为后端分页工作流库：搜索、排序、page/pageSize/total、当前页全选与跨页选择、单项/批量删除均已接通。删除使用 revision+sourceHash CAS；Schedule、launcher item 和 queued/running Run 采用引用阻止策略，批量返回逐项结果，历史 Run journal 保留。4 个 Go 包、17 个前端测试、1546-key i18n 与 typecheck 通过，正式绑定为 14 services / 102 methods。
- Stage 4 Slice 11 已把 exact provider-native endpoint 纳入 AI Model Profile v2：官方 Responses/Messages 地址为显式默认，自定义 HTTPS 或显式风险确认的 loopback HTTP 会进入 profile/evaluation/consent identity；TestProfile 与安装运行复用同一 sealed endpoint。生产 HTTP client 禁用环境代理与 redirect，API key 仍只在 secure store。AI/services Go 测试、6 个前端测试、1552-key i18n 与 typecheck 通过。
- Stage 4 Slice 12 已恢复 launcher 可发现性：主标题栏与 launcher 设置页可直接打开，设置页显示 system.launcher-toggle 的 active/unbound/failed 状态；重复打开聚焦既有窗口。空 launcher 可通过 OpenLauncherSettings 显示/聚焦主窗并由主壳安全导航到配置，不强制重载编辑器。tools/desktopapp Go 测试、16 个前端测试、1562-key i18n 与 typecheck 通过；正式 Windows window/hotkey smoke 留到当前 Stage 4 门禁。
- Stage 3 WebView smoke 同时修复两个真实时序问题：空白连线落点探针改为画布网格扫描；调试 RPC 返回与事件推送统一按 monotonic generation 合并，防止 completed 被旧 running 快照覆盖。
- 新旧对比固定使用 9fce7870^ 的旧路由、视图、store、编辑器 composables、Wails 调用与 runtime/service，对照当前入口、管理、创作和运行四层。
- Stage 4 整批门禁已通过：task check 覆盖 Go、144 个前端测试、AI eval、65.6% 全局覆盖率、静态检查、i18n、RPC 契约和 production bundle；task build 生成正式 bin/Yotta.exe；隔离 Windows WebView smoke 实际创建工作流、点击/拖放到 3 个画布节点，确认 103 个目录节点、AI review、资源/录制入口，并人工检查编辑器与资源页 PNG。文件选择取消、F9 捕获和 launcher 聚焦行为由宿主边界测试覆盖，未触碰用户 data。
- Stage 5 Slice 13 已完成：durable schema 从 automation.win32Targets 迁移到带 targetKind/adapterKind 的 automation.targets，旧数据迁移会撤销 v1 consent；Workflow/节点使用语义 desktop-window，Win32 是 Adapter identity。installed 模块通过 descriptor + registry 提供小 Interface，Win32 与 test Adapter 走同一 conformance；provider 的标准输入/截图复用 automation/controller，policy/bootstrap/前端由 descriptor 驱动。Linux/amd64 与 darwin/arm64 core test binary 交叉编译、task check、task build 与 Windows WebView smoke 均通过。
- Stage 6 Slice 8 已完成：Android exact ADB identity、package、consent、设备发现与健康通过同一 automation.targets/Adapter/provider/admission seam 接入；Settings 支持 Windows/Android 安装，工作流创建提供通用/Windows/Android/跨目标引导模板。通用截图、点击、移动、拖拽、滚动、文本与激活节点支持 Android，stop-target-app 仅向 android-device 开放；组合键、相对移动与低级录制回放在 Catalog/admission 层禁用。task check、task build、Linux/amd64 与 darwin/arm64 cross-compile、Windows WebView smoke 和 bilibili_api35 ADB emulator smoke 全部通过；104 个目录节点与视觉截图已检查。
- Stage 7 已完成：工具栏与 Ctrl/⌘+F 恢复 Source-native 节点定位；旧 command palette 由现有入口替代。旧 Container zip、任意宿主 JS/yt console 与语义 reroute node 明确不恢复；Source portability、资产规模化、完整 subgraph authoring 与 Browser CDP 进入诚实的 post-3.1 产品边界。
- Stage 7 整批门禁全绿：task check（145 前端测试、1595 i18n keys、65.4% Go 总覆盖）、task build、Windows WebView smoke（104 catalog nodes/3 canvas nodes）与编辑器/资源页截图目检均通过。

## Open questions

- 删除工作流时，关联 schedule/launcher item 应阻止删除、要求显式级联，还是自动失效；历史 Run journal 默认应保留。
- 自定义 AI 地址首期只支持 provider-native Responses/Messages，还是新增显式 OpenAI-compatible Chat Completions provider；不得静默协议回退。
