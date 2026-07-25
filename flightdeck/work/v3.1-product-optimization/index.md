# 3.1 产品创作体验与运行工作台优化

## Goal

在稳定的 3.1 架构上恢复专业工作流创作能力，修正编辑器状态表达、节点发现、资源编辑和运行工作台体验。

## Status

Open

## Current

Stage M3b 完成：Installation-local target profile、secure credential logical binding、精确 run/schedule
consent、Installation-ID 执行和设置界面均已闭环；secret 始终只留在本机安全存储，Readiness 每次从当前
目标、依赖与 credential availability 重新计算。

## Next

开始 [Stage M4](plan.md)：定义 data-only `.yotta-workflow`、统一 Installation Plan、离线包与
Node Package trust/install/consent 边界。

## Progress

- 2026-07-26 M3b 完成：Workflow Credential Requirement 以 kind + logical binding ID 选择当前 AI
  credential profile，安全存储只投影 available 状态且不返回 secret；未知、类型不兼容、已删除或不可用
  binding 在更新和 Readiness 两处 fail closed。设置 Modal 支持选择兼容 credential、失效引用与无候选修复入口，
  Wails 契约更新为 17 服务/155 方法/229 模型。`task check` 通过 35 个 Go 包与前端 83 文件/353 项测试；
  WebView smoke 暴露并修复 runner 的长异步 quick-add Promise 被 WebView2 回收问题，稳定回归测试与
  `20260726-060626` 真实旅程均退出 0。
- 2026-07-26 M3b Target Profile 完成：首次安装从 exact Release Definition materialize 独立 profile，
  schema 6 持久化 canonical settings 与 Target Installation ID，schema 5 记录按 generation CAS 惰性补齐。
  Readiness 只接受仍安装、类型/adapter/version 精确匹配且已授权的自动化目标；Wails 154 个方法暴露
  设置读写，工作流列表提供安装级状态、运行授权、执行与密集设置 Modal，凭据区明确只显示绑定状态。
  `task check` 通过 36 个 Go 包和前端 83 文件/353 项测试；Windows WebView smoke
  `20260726-054225` 退出 0，已安装列表与设置截图已目检。
- 2026-07-26 M3a 首批核心完成：verified Release projection、多实例 Installation 与独立 Readiness
  深模块进入 Content Catalog schema 4 和桌面/CLI composition；Release/Installation 原子提交、identity
  collision 回滚、五类并存 blocker、run/schedule 分离与精确 Release consent lineage 已由测试锁定。
  `task check` 通过 38 个受影响 Go 包和 82 个测试文件/351 项测试。
- 2026-07-26 M3b 配置事实源完成首批：Catalog schema 5 持久化 generation-CAS target/credential
  logical binding 与精确 Release run/schedule consent；Readiness 改从 repository 与 verified/enabled/
  host-compatible Node Package projection 读取，不接受调用方伪造本机状态。`task check` 通过 36 个受影响
  Go 包。
- 2026-07-26 M3b 执行入口完成：`PrepareExecution` 从一次持久事实快照产出 opaque immutable execution，
  手动运行与 Schedule 均使用 Installation-ID、独立 scope consent、exact Release Source 和同一 Application
  compiler/admission/Run Ledger 路径。Schedule schema 2 保存 Installation reference；schema 1 迁移时强制
  disarm 等待显式修复，启用保存与触发时双重 fail closed。Wails/前端改为列出 Installation，计划启用可显式
  授予 schedule consent；`task check` 通过 35 个 Go 包及前端 82 文件/351 测试，Windows WebView smoke
  `20260726-045653` 退出 0，计划、恢复、编辑器与资源库截图已目检。
- Stage A–I 完成专业画布、Source-native 子图、Macro/InputClip、资源工作区、typed Authoring Surface、
  durable Snippets、Tab 快速添加、Macro 原地编辑、共享计划 Modal 与三路安全退出。
- Stage J 闭环录制/模板连续性、运行停滞可解释性、节点与大选项发现、参数密度、Run State 初值和动态
  Switch；阶段完整门禁与 WebView smoke 通过。
- Stage K 修正 Catalog 图标、运行工作台、可缩放/折叠侧栏、durable 状态类型、动态端口标题和悬空草稿
  执行语义，并把日常门禁改为按 Git 变更路由。
- Stage L 统一资源列表与拖拽，修正 binding 清除、日志 message、Inspector 总开关、工作流设置和子图
  空态/接口推断，所有事实继续写入 Source/Contract/runtime。
- 录制链已统一为后端权威 `armed → countdown → recording`，默认 F10/F11/F12；校准按“目标自定义 >
  活动档案 > 未校准”解析，保存页、任意时间线裁剪和可创建资源元数据均已闭环。
- 精准回放恢复绝对单调 deadline 和 session 级目标固定，按录制源/本机目标 counts/360 自动换算；撤回
  `turn-scale` 并把开发期契约安全迁回稳定摘要，production bundle 为 218155/220000 bytes。
- 2026-07-21 收尾增量门禁通过：Wails/节点/工作流契约、25 个相关 Go 包及 vet、frontend
  format/lint/typecheck/i18n、70 个测试文件/294 项测试；ESLint 债务从 24 收紧到 23。未运行无关 Rust、
  全仓 coverage/staticcheck；本轮 production bundle 已在精准回放修复后单独通过。
- 2026-07-21 精准录制/回放 Windows 真机校验通过，Stage L 验收闭环。
- Stage M 已记录下载工作流的依赖预检与本机 target/credential 重绑定；现有 Bundle 携带 Source/Blob，
  但在 M1–M3 完成前不宣称下载工作流可无提示直接运行。
- 2026-07-21 确认在线试验市场使用独立兄弟仓库 `yotta-registry`、Nuxt UI 与 GoFrame；实现基线已对齐
  `yueli-official/platform/flightdeck/knowledge` 的 API、SSR/BFF、目录站、OIDC 与 staged import 约定。
- 2026-07-21 Stage M Grilling 已收敛 Workflow Resource/Global Asset、Release/Installation、Target/Credential
  Profile、Node Package trust/install/consent、更新/派生/回退、Registry 上架/下架、作者证明与在线/离线交付语义，
  并重写为 M1–M10 跨三仓执行计划。
- 2026-07-21 M1 完成首个正式可移植 Source/Bundle 合同：新增 Workflow Resource、Resource Binding、Target Profile
  Definition、Credential Requirement 与精确 Node Package Dependency；schema 统一负责严格校验、资源解析和 Blob
  inventory，compiler 校验 Catalog lock 并复用原 Blob value path，Bundle manifest 锁定 dependency 且包含所有资源字节。
- M1 `task check` 通过：Workflow/Wails 合同一致，20 个受影响 Go 包通过，frontend format/lint/typecheck/i18n 通过，
  70 个测试文件/294 项测试通过；未运行非增量的 `task check:full`、真实宿主或 production bundle。
- 2026-07-24 Stage N 启动：折叠开放边界修复已提交为 `26577e20`；调用节点和内部 boundary 的 handle gutter/
  长标签布局已由组件测试锁定；Unreal、Blender、Node-RED、Unity 官方资料已收敛为定义/调用分层、canonical
  interface 清单和安全定义删除方案。
- 2026-07-24 N1 完成：新增 Source-derived 子图管理器、同名 ID 消歧、调用引用定位、接口健康摘要和定义/
  调用无歧义删除；修复缺失 graph ID 误删最后一个定义。`task check` 通过 74 个测试文件/304 项测试。
- 2026-07-24 N2 完成：Graph interface 分离稳定 ID/显示名称，显式发布单入口、typed data ports 和命名
  exec/error exits；自动推导改为引用安全预览。`task check` 通过 21 个受影响 Go 包和 76 个测试文件/310 项测试。
- 2026-07-24 N3 完成：GraphCall 复制、定义复制/分叉、调用展开和显式原子级联删除进入统一 EditorSession/
  authoring patch；展开保留 value/default/blob/resource binding，并新增正式 `bind-resource` 合同。`task check`
  通过 12 个受影响 Go 包和 77 个测试文件/318 项测试。
- 2026-07-24 N4/Stage N 完成：真实 service/store 覆盖保存重开、展开、物理级联删除和 compiler；WebView
  smoke 修正冷启动、选择、推导确认与调试终态竞态后退出 0，子图截图无裁切/遮挡。production editor gzip
  205329/220000 字节，`task build` 生成版本 3.1.0 Windows GUI `bin/Yotta.exe` 并通过隔离启动 smoke。
- 2026-07-24 O1/Stage O 完成：修复复合状态初值的 reactive proxy `DataCloneError`，由 Authoring
  Projection 提供 schema 验证后的权威初值；19 类状态通过 UI、Source 保存重开、compiler 和 runtime
  矩阵。按键组合列表可由通用状态节点读写，Increment 继续只接受 numeric；非法 JSON 初值显示中文行内
  错误。`task check` 通过 42 个 Go 包和 79 个前端测试文件/339 项测试，Windows WebView 文件元数据
  旅程通过，production editor gzip 205314/220000 字节。
- 2026-07-24 P1/Stage P 完成：真实 `fishing-v2` 重组为准备、拉钩、溜鱼、结算、买饵、卖鱼六个
  Source-native 子图；修复 Repeat/ForEach/Retry 区域控制输入被折叠器误判为 callable graph entry。
  原 revision 4 已移至不参与 SourceStore 扫描的 `bin/data/backups/workflow-sources/`，当前 revision 8 由 production CLI 编译 0 诊断；
  Windows WebView 管理器和六个子图逐项打开，无不健康接口、alert 或 JS error。`task check` 通过
  12 个受影响 Go 包，`task build` 生成当前 3.1.0 GUI 并通过隔离启动 smoke。
- 2026-07-25 Q1/Stage Q 完成：左侧 rail 重组为子图、Macro、精准录制、视觉模板和 Snippet；子图管理
  从顶部 Popover 改为停靠面板，节点发现收敛到 Tab 与显式“添加节点”。修正 WebView 烟测残留目录选择器和
  子图新建稳定入口；`task check` 通过 79 个前端测试文件/340 项测试，`task webview:smoke` 退出 0，
  六类关键 PNG 已目检。
- 2026-07-25 M2a 完成：Workflow Resource 增删改/引用保护进入正式合同与 EditorSession；资源侧栏提供
  scope/mode、三筛选、数字分页、逐项/本页/跨页选择、元数据与批量删除；Global Asset 按钮/双击创建
  图片、Macro 或 InputClip snapshot。`task check` 通过 80 个前端测试文件/343 项测试，最终 Windows
  WebView smoke 退出 0；同时根治首轮 L 形框选误选和 Ctrl modifier 假多选。
- 2026-07-25 修复 M2a production bundle 回归：`WorkflowResourceDock` 从 editor 同步依赖改为按需 chunk，
  editor 初始 gzip 223692 → 201936 bytes；`task build` 退出 0，`bin/Yotta.exe` 通过 Windows GUI metadata
  与 5 秒隔离启动 smoke，随后 `task check` 再次通过 13 个 Go 包和 80 个前端测试文件/343 项测试。
- 2026-07-25 根据真机反馈撤回资源侧栏“使用/管理”双模式：范围切换增加显式 primary 激活态，列表同时
  提供使用、选择和 overflow 管理，筛选列等宽铺满。WebView smoke 新增 computed-style/实际宽度断言并
  分别截图两种 scope；同时修复 WebView2 仅监听 IPv6 `::1` 时脚本误轮询 IPv4 导致的首轮假失败。
- 2026-07-25 资源预览/定位闭环：资源库、选择器和节点模板缩略图统一支持大图、适应窗口、实际尺寸和
  25%–400% 缩放；所有通用素材绑定字段可按 Workflow `resourceId + variantId` 或 Global Asset GUID
  打开左侧对应面板并精确过滤、高亮目标。1000 条素材 GUID 查询、`task check`（29 个 Go 包、81 个
  前端测试文件/347 项测试）、production bundle（editor 201985/220000 gzip）及持续等待同一进程退出码的
  WebView smoke 均通过；同时补齐 M2 资源 authoring 命令的 Wails RPC 合同快照。
- 2026-07-25 R1 完成：审计发现 GUI settings/data 绑定 exe 目录、日志依赖 cwd，Store 启动全量扫描、
  损坏/耐久语义分裂，Run journal 累计重写以及 CAS 缺少持久化引用/宽限 GC。结合 Microsoft、Go、
  SQLite、Git/OCI/containerd 66 个一手链接，目标确定为两个事务一致性域加文件 CAS；根布局与产品版本
  分离，正式迁移具备 checksum registry、dry-run、backup、verify 和 resume/recovery。
- 2026-07-25 R2 完成：新增 `internal/storage` RootSet、Windows Known Folder resolver、root manifest、
  跨平台 writer lease、只读 health 和确定性 migration registry；settings 使用 schema/generation/checksum
  envelope 并从 primary/backup/staging 选择最新完整代。GUI/CLI/dev/build smoke 不再依赖 exe/cwd，
  `workspace-3.1` 隐式 rename 已删除，Windows 启动失败会显示错误框。
- R2 验收：`task check` 由持续后台 wrapper 保留同一进程最终退出码并返回 0（30 个受影响 Go 包）；
  `task build` 返回 0，3.1.0 Windows GUI metadata 正确；修正后的 isolated smoke 将 app/profile 分离、
  断言 exe 旁无 data/settings，存活 5 秒后清理退出 0；production CLI `health` 返回受支持 layout 1。
- 2026-07-25 R3 完成：锁定 CGO-free `modernc.org/sqlite v1.54.0`，建立 `content.db`/`runs.db` 的
  application ID、schema 1、不可变 checksum migration ledger、WAL/FULL、quick/foreign-key health 和
  Online Backup；故障注入、损坏/漂移、活动 WAL snapshot、CGO-off 与 race 测试通过。
- R3 验收：持续后台 `task check` 同一进程退出 0；`task build` 退出 0并通过 Windows metadata/隔离
  5 秒 GUI startup smoke；production CLI 隔离 profile 创建两个数据库，health 返回两库 healthy 且路径脱敏。
- 2026-07-25 R4 完成：Content Catalog schema 2 接管 Global Asset、variant/tag、object ref/lease/GC；
  CAS layout 2 改为两级分片且启动配额来自 Catalog，不扫描/哈希全部对象。CAS publish 后才能事务提交引用，
  缺失/非 active 对象 fail closed，失败提交留下的孤儿由宽限 GC 回收。
- R4 验收：10k Asset、100k object inventory、256 MiB 固定缓冲、dedup/冲突、并发 revision、共享
  Workflow Blob、宽限/lease/pin/stale preview 与 staging/trash/Catalog failure fixture 通过；相关 race、
  `task check`（bindings + 48 个 Go 包）、`task build`、隔离 desktop startup 和 WebView asset smoke 均退出 0。
- 2026-07-25 R5 完成：Content Catalog schema 3 接管 Workflow Source canonical bytes、metadata、revision、
  Blob refs 与 quarantine；提交只引用 active CAS inventory 并在失败时完整回滚。Program cache layout 2
  按 compiler build + Node Catalog identity 分代，支持 count/byte quota、持久 LRU、stale/corrupt rebuild。
- R5 验收：1,000 Source 查询、CAS 引用回滚、quarantine repair、Program identity/LRU/corruption fixture 与
  相关 race 通过；最终 `task check` 同一后台进程退出 0，`task build` 与隔离 Windows startup 通过，
  production CLI health 返回 Content schema 3、Run schema 1 与两库 healthy。
- 2026-07-25 R6 完成：Run Ledger schema 2 接管 summary/event/value；domain Record 继续负责 generation、
  digest、状态机与 journal 顺序，Repository 只原子追加 event/更新 bounded head。timeline page 使用一致
  SQLite snapshot，archive/purge 与 payload CAS roots 已闭环；legacy v1 JSON 幂等导入且不删除旧 bytes。
- R6 验收：全 `internal/...` 普通测试、Catalog/Run race、最终 `task check`（30 个受影响 Go 包）、
  `task build`、Windows metadata/隔离 startup 均退出 0；production CLI health 返回 Content schema 3、
  Run schema 2、WAL/FULL、quick-check ok 与两库 healthy。
- 2026-07-25 R7 核心实现完成：冻结 released layout 1 fixture，layout 2 migration Module 建立只读 plan、
  空间估算、完整 snapshot、immutable journal/checksum、resume/rollback、legacy Run quarantine/restore
  与脱敏诊断导出；CLI 和启动期独立 recovery window 复用同一 Interface。prepared/Catalog/manifest
  前后 kill-point、journal/snapshot 篡改和 GUI handler 定向测试通过。
- R7 验收：targeted race、最终 `task check`（router self-test、AI eval、bindings、32 个 Go 包）、
  `task build` 均退出 0；新增 `task smoke:storage-migration` 用 production 二进制验证 plan read-only、
  invalid Run recovery、GUI kill、quarantine/resume、layout 2 双库 health 与迁移后 GUI 存活。
- 2026-07-25 M2b 完成：新增 Workflow Resource authoring Module；编辑器录制/截图直接写 CAS 并返回完整
  图片/Macro/InputClip resource，资产库仍创建 Global Asset；本机素材拖放改为 GUID-only payload 后从
  Catalog 重新取权威 metadata、创建 snapshot 与 Resource Binding。`task check` 通过 34 个 Go 包和前端
  82 文件/351 测试；`task build` editor gzip 202909/220000，Windows 隔离启动 smoke 退出 0。
- 2026-07-26 M2c 完成：三类 Workflow Resource 可显式提升为独立 Global Asset，提升前严格验证 resource
  与全部 CAS BlobRef，重复提升复用内容字节但分配不同 GUID；资源侧栏刷新素材库且不产生 Source dirty/undo。
  `task check` 通过 36 个 Go 包和前端 82 文件/351 测试；`task build` editor gzip 202920/220000，
  Windows GUI metadata 与隔离启动 smoke 退出 0。
- 2026-07-26 M2d/Stage M2 完成：`resourceauthoring` 直接从 Workflow Resource/CAS 打开 Macro 与
  InputClip，提供严格 carrier/metadata 校验、分页事件、动作重写、精准裁剪与三类显式 duplicate；
  authoring `replace-resource` 保留 ID/kind、共享 binding 和单 undo。资源侧栏显示三类 Source-native
  摘要，并复用现有 Macro/精准录制工作台完成内容编辑，不查询 Global Asset。
- M2d 验收：最终 `task check` 路由 113 个变更文件，AI 8/8、Wails 17 服务/148 方法、37 个 Go 包、
  前端 82 文件/351 测试全部通过；`task build` editor gzip 203779/220000 并通过 Windows metadata/
  隔离启动。修正 WebView smoke 的旧文件夹恢复夹具为 Catalog quarantine 后，`task webview:smoke`
  退出 0，`20260726-010939` 的恢复面、编辑器、资源工具、资源库与计划截图已目检。

## References

- [Real-device feedback](references/real-device-feedback-2026-07-20.md) — 正在收集的真机问题、证据与待验证项。
- [Real-device feedback follow-up](references/real-device-feedback-2026-07-21.md) — 第二批八项反馈、根因与复测点。
- [Real-device feedback batch 3](references/real-device-feedback-2026-07-21-batch-3.md) — 第三批九项反馈与验收点。
- [Real-device feedback research](references/real-device-feedback-research-2026-07-20.md) — 十项反馈的本仓事实映射与一手 UI/交互资料。
- [3.0/3.1 editor audit](references/current-vs-3.0-editor-audit.md) — 能力连续性和不恢复旧 runtime 的依据。
- [Editor discovery and modal decisions](references/editor-discovery-and-modal-decisions.md) — Stage I 研究与取舍。
- [Node density and optional pins](references/14-node-density-and-optional-pins.md) — 紧凑节点投影证据。
- [Workflow resource editing](references/15-workflow-resource-edit-and-safe-exit.md) — Macro 编辑与退出语义。
- [Tab and Snippet flow](references/16-tab-menu-and-snippet-shortcuts.md) — 快速添加与快捷键实现边界。
- [Schedule modal flow](references/17-schedule-modal-flow.md) — 计划 Modal 的验收记录。
- [Canvas authoring boundary](../../knowledge/frontend/canvas-node-authoring-boundary.md) — 当前画布创作规则。
- [Build and acceptance](../../knowledge/build/build.md) — 完整门禁的触发条件。
- [Subgraph management research](references/subgraph-management-research.md) — 定义/调用、接口编辑和生命周期的一手资料与本地方案。
