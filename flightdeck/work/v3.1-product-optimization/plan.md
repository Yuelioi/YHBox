# 3.1 产品优化计划

## Outcome

在唯一 3.1 Source/compiler/runtime 上持续改善专业创作与运行体验，同时保持每个 Stage 都能由真实用户旅程、定向验证和阶段门禁闭环。

## Current stage

Stage A–K 已完成实现与增量门禁。Stage L 已完成实现与定向门禁，等待 Windows 真机复测；Stage M
记录网上工作流分发的后续闭环，不在本次真机修复中仓促扩展。

## Stage L — 第三批真机反馈闭环

范围：修复 `FD-19` 至 `FD-28`，保持资源、binding、Workflow Source/metadata 与子图接口的唯一事实，
不以临时前端状态伪造成功。

- [x] L1 — 资源拖拽、录制恢复与统一资源列表（FD-19、FD-20、FD-24）
- [x] L2 — 输入清除语义、模板 binding 与日志 message（FD-21、FD-22、FD-23）
- [x] L3 — Inspector 总开关与工作流设置/更多工具栏（FD-25、FD-26）
- [x] L4 — 新建子图 boundary 布局、非阻塞空态与接口刷新前置条件（FD-27）
- [x] L5 — 精准录制继承活动鼠标校准档案并明确目标覆盖语义（FD-20 真机复测）
- [x] L6 — 精准录制保存、鼠标裁剪、运行期校准与失败解释闭环（FD-28）
- [x] L7 — 任意裁剪边界状态补全与可创建资源元数据（FD-28 真机复测）
- [x] L8 — 录制元数据、工作流设置与 AI 审查按需加载，恢复 editor bundle 预算
- [x] L9 — 简易/精准录制统一待开始、3 秒倒计时与 F10/F11/F12 快捷键（FD-29）
- [x] L10 — 恢复精准回放绝对时间轴并把目标解析收敛到 playback session（FD-30）
- [x] L11 — 展示 InputClip 录制源/本机目标 counts/360，并按两份校准自动换算（FD-31）
- [x] L12 — 撤回临时转向倍率契约，清理开发期摘要并修复派生编译诊断

阶段门槛：每个切片先建立捕获截图症状的定向测试；完成后只运行增量 `task check` 与对应 WebView/
真实宿主旅程，不运行无关 Rust、全仓 coverage 或 production bundle。

## Stage M — 网上工作流分发与本机绑定

范围：让下载的 `.yotta-workflow` 在执行前明确知道依赖什么，并安全绑定当前机器的目标、凭据与校准；
不把 HWND、设备 ID、凭据或本机 counts 写进可移植工作流参数。

- [ ] M1 — Bundle 清单暴露精确 Node Package/NodeRef 依赖并在导入前完成兼容性预检
- [ ] M2 — 导入后提供逻辑 target/credential 到本机 installation 的显式重绑定流程
- [ ] M3 — 缺失 package、未登记契约迁移和未校准目标给出可操作诊断，禁止静默替换语义

阶段门槛：跨两套独立 workspace 导出/导入包含 InputClip 的工作流；Blob 身份和源 counts 保持不变，
作者机器的本地目标/凭据不泄漏，接收方未完成绑定前不得误运行。

## Stage K — 第二批真机反馈闭环

范围：修复 `FD-11` 至 `FD-18`，继续以 Catalog/Projection、Workflow Source、Compiler Program 和统一
runtime 为唯一事实；让调试中的未连接草稿节点不阻塞可达执行链。

- [x] [K1 — 图标、Switch、工作台与侧栏](slices/stage-k1-real-device-follow-up.md)（FD-11、FD-12、FD-14、FD-16）
- [x] [K2 — Run State、节点选择与草稿执行](slices/stage-k1-real-device-follow-up.md)（FD-13、FD-15、FD-17、FD-18）

阶段门槛：Catalog 图标、动态端口标题、状态类型、运行工作台和悬空草稿分别有定向回归测试；完成后
运行 `task check`，再在 Windows 真实宿主逐条复测八条用户旅程。

## Stage J — 真机反馈修复闭环

范围：只在唯一 3.1 Source/Contract/Compiler/runtime 上修复 `FD-01` 至 `FD-10`，复用现有
Authoring Projection、资源与运行事实，不恢复 3.0 Container 或第二套状态路径。

- [x] [J1 — 录制与模板上下文连续性](slices/stage-j1-recording-template-continuity.md)（FD-01、FD-02）
- [x] [J2 — 运行停滞可解释性](slices/stage-j2-runtime-observability.md)（FD-04）
- [x] [J3 — 节点与选项发现](slices/stage-j3-discovery-and-selection.md)（FD-03、FD-08、FD-10）
- [x] [J4 — 参数密度、行内编辑与 Run State 初值](slices/stage-j4-authoring-density-and-state.md)（FD-05、FD-06、FD-09）
- [x] [J5 — 动态 Switch 稳定分支拓扑](slices/stage-j5-dynamic-switch.md)（FD-07）

非目标：不把真机反馈拆成无关的视觉换肤；不改变许可证、发布范围或旧工作流兼容承诺；不以
前端临时状态伪造运行、资源或动态端口事实。

阶段门槛：每个 J 子项先建立能捕获用户症状的定向测试；J1–J5 全部通过后运行 `task check`，并按
改动触发 Windows WebView/真机 smoke。若某项缺少自动化 seam，必须在对应 Slice 记录人工验收步骤。

## Starting the next stage

只有出现新的真机反馈或用户明确扩展产品范围时才创建下一 Stage：

1. 先复现一条具体用户旅程，记录当前行为、期望行为和可核验差异。
2. 核对 [context](context.md) 中的架构边界，确认修复不会恢复 3.0 Container、第二套 store 或第二套 runtime。
3. 把同一旅程上的相邻问题组成一个可交付 Stage，并在这里写明范围、非目标与验收门槛。
4. 实施中运行最小定向检查；Stage 完成后统一运行 `task check` 和被改动触发的真实宿主 smoke。
5. 验收后重写 `index.md` 的 Current、Next 与 Progress，让下一会话只看到仍然成立的状态。

## Stable constraints

- Selection、execution、debug 和 validation 保持独立状态。
- 复杂节点由 Authoring Projection 与类型级 Editor Adapter 承载，画布只显示高频摘要。
- Macro 与 InputClip 分轨；脏资源退出必须保留取消、放弃、保存并退出三路语义。
- 单对象短流程优先 Modal；长生命周期、多页面任务才使用独立路由。

## Historical evidence

- [Approved plan through Stage G](references/approved-plan-through-stage-g.md)
- [Stage H node menu and template flow](references/13-node-context-menu-and-template-flow.md)
- [Stage I node density](references/14-node-density-and-optional-pins.md)
- [Stage I resource editing](references/15-workflow-resource-edit-and-safe-exit.md)
- [Stage I Tab and Snippet flow](references/16-tab-menu-and-snippet-shortcuts.md)
- [Stage I schedule modal flow](references/17-schedule-modal-flow.md)
