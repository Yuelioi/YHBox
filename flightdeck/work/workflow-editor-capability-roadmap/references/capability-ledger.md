# Yotta 3.1 capability ledger

## Purpose and authority

本账本是 3.1 发布前能力事实源。它以旧版真实行为为 oracle，但只接受 3.1 唯一 Source → Compiler → Program → Executor 路径上的实现。

能力状态不能由文件存在、节点数量、单元测试、页面截图或 production build 单独推导。每项必须依次闭合：

1. `E` entrance：用户能发现、进入、取消并从失败恢复。
2. `M` management：能创建、查看、修改、删除，并在规模增长后继续使用。
3. `A` authoring：Source/Inspector 绑定稳定 ID、slot、BlobRef 或 typed value。
4. `R` runtime：Catalog、compiler、admission、provider、journal 使用同一契约。
5. `J` journey：真实宿主或可信端到端 fixture 证明用户任务成功。

状态：`verified` 表示五层证据可信；`reverify` 表示已有实现但缺纵向证据；`rebuild` 表示边界需要重构；`restore` 表示能力缺失；`remove` 表示明确不恢复；`defer-decision` 表示等待旧真实 workflow 证据，但不得用来阻塞已确定的 P0 工作。

## Core authoring and execution

| ID | 能力 | 3.0 行为 oracle | 当前事实 | 3.1 决策 | Owner / gate |
| --- | --- | --- | --- | --- | --- |
| CORE-01 | 新建工作流自动提供 Start | Container 创建后可直接执行 | clean WebView workspace 自动创建唯一 Run Start 并可继续创作 | verified | Editor Authoring / G01 |
| CORE-02 | 节点目录搜索、拖入、删除 | 目录与 NodeSearchModal | 145-node catalog 可搜索；单选和多选 Delete 均进入同一 Source command | verified | Editor Authoring / G01 |
| CORE-03 | 拖线提示与上下文候选 | inline node candidates | connection plan、候选和 compiler 使用同一 type relation fixture | verified | Projection / G01、G10 |
| CORE-04 | 单击、拖线时节点不误移 | 旧 canvas 手势 | EditorSession 分离选择、拖动和连接命令，WebView 创作旅程通过 | verified | Editor Session / G01 |
| CORE-05 | 多选、复制、对齐、分布、排序 | selection/layout/snap 工具 | selection toolbar、复制、对齐、分布、自动布局和多选删除通过组件与 WebView 旅程 | verified | Editor Commands / G01 |
| CORE-06 | 强类型端口与显式转换 | 旧 Expr/弱类型 fallback | nominal TypeRef、generic solving、conversion graph 与 compiler parity 全绿 | verified | Type System / G10 |
| CORE-07 | Run State 搜索、拖出读取/写入 | VarsPanel 可直接参与创作 | 1000 states 可搜索/分页，并可直接创建 Read、Write、LastChange、Increment | verified | State Authoring / G10 |
| CORE-08 | Repeat index 等输出可消费 | 旧节点输出可连线 | Integer 输出能发现数值比较、转换与日志链；typed candidates/compiler parity 通过 | verified | Projection / G10 |
| CORE-09 | compile diagnostics 与定位 | ProblemsBar | compiler diagnostics、节点定位与 WebView compile journey 通过 | verified | Compiler/UI / G01 |
| CORE-10 | 断点、暂停、单步、继续 | 旧 DebugPanel | 同一 scheduler debugger 的断点、暂停、单步和继续通过 WebView 旅程 | verified | Executor/UI / G11 |
| CORE-11 | Run timeline 与结构化失败 | 旧 debug/log panel | typed RPC envelope、journal timeline 与结构化失败恢复纳入 R1–R3 gates | verified | RPC + Run UI / G11 |
| CORE-12 | Source-native 多图、GraphCall | 旧 subgraph | typed interface、GraphCall、导航/调试和 AI review WebView 旅程通过 | verified | Multi-graph / G12 |

## Workflow and installation management

| ID | 能力 | 3.0 行为 oracle | 当前事实 | 3.1 决策 | Owner / gate |
| --- | --- | --- | --- | --- | --- |
| LIB-01 | 工作流运行、编辑、删除 | 列表操作完整 | WebView 列表创作/运行与删除交互、损坏 Source 隔离恢复均通过 | verified | Workflow Library / G09 |
| LIB-02 | 批量选择、删除、分页、搜索 | 旧库管理 | 真实 workflow service 1000-source fixture 验证搜索、排序和有界分页 | verified | Workflow Library / G09 |
| LIB-03 | Source import/export 与 exact blobs | 旧版仅 Container export | canonical Source 与 referenced Blob 从独立 Source/Blob Store 导出，导入第二套 clean stores 后 exact bytes/digest 存在 | verified | Portability / G12 |
| LIB-04 | 损坏/缺失引用可恢复诊断 | 旧数据常以崩溃暴露 | 坏 Source 可隔离修复/删除；stale Program cache 自动重建；当前 workspace 无需删 data | verified | Workspace Load / G13 |
| AI-01 | AI endpoint/API URL 属于安装属性 | 旧设置可配置连接 | 自定义 loopback API URL + credential binding 走 Source→Compiler→Admission→native provider→journal；Source 不含 endpoint secret，服务端收到 exact path/auth | verified | AI Installation / G14 |
| LAUNCH-01 | 悬浮窗启动入口 | FloatingLauncher 可见 | 主壳入口可发现；真实 WebView 执行安全 workflow 成功，隐藏后复用同一窗口且无重复 target | verified | App Shell / G15 |

## Desktop automation and authority

| ID | 能力 | 3.0 行为 oracle | 当前事实 | 3.1 决策 | Owner / gate |
| --- | --- | --- | --- | --- | --- |
| AUTO-01 | 安装应用后建立稳定身份 | executable digest 安装 | 真实 HTGame identity + versioned target profile + manifest generation 运行成功 | verified | Manifest + Windows / G02 |
| AUTO-02 | 捕获窗口同时安装应用、目标和 consent | F9 捕获后可使用 | F9 临时 hook、exact metadata、取消释放与同进程 generation publish 已闭合 | verified | Target Runtime / G02 |
| AUTO-03 | exact title 原样保存 | 精确标题支持尾随空格 | fixture 与真实异环标题末尾两个空格均逐字符匹配成功 | verified | Windows Adapter / G02 |
| AUTO-04 | 显式 RE2 regex | exact/regex 两种模式 | native 动态标题显式 regex 匹配，exact/regex 无隐式互换 | verified | Windows Adapter / G02 |
| AUTO-05 | 多窗口确定性选择 | 用户可绑定目标窗口 | unique 多匹配返回 target-ambiguous；关闭重开重新解析 | verified | Windows Adapter / G03 |
| AUTO-06 | F9 临时全局热键 | 捕获时可跨应用触发 | native low-level hook 捕获前台 exact metadata，完成/取消释放 session | verified | Capture Session / G02 |
| AUTO-07 | target 修改/删除同进程生效 | 旧版无启动快照割裂 | generation publish/rollback、old-Run lease、idle reclaim 与依赖删除闭合 | verified | Target Runtime / G04 |
| AUTO-08 | consent 可批量但不静默授权未来项 | 旧授权体验较直接 | snapshot 批量授权闭合；profile/manifest 漂移仅撤销 stale consent 并保留安装 | verified | Policy UX / G04 |
| AUTO-09 | Press Keys/Type Text | 按键节点支持快捷录入 | native F8/字符投递与真实异环 ESC Run journal 成功 | verified | Manifest + Windows / G05 |
| AUTO-10 | click/move/drag/relative move | 基础鼠标自动化 | native click/move/relative/scroll/SendInput drag 生效，drag 进入真实 hook | verified | Windows Provider / G05 |
| AUTO-11 | held key/mouse 保证异常释放 | KeyHold/MouseHold | Hold Keys/Hold Pointer/Release Held 使用 Run-owned HandleRef，teardown native 释放 | verified | Input Lease / G05 |
| AUTO-12 | 激活、等待、关闭、置前、移动缩放、窗口状态 | 完整窗口节点族 | activate/close/move-resize/state/set-state/wait-present/wait-gone 与 native 旅程闭合 | verified | Window Operations / G06 |
| AUTO-13 | 普通与管理员窗口 | 自动化软件默认 UAC | production manifest 固定 requireAdministrator；普通 fixture 与真实 UnrealWindow 路径有证据 | verified | Windows Host / G02–G06 |
| AUTO-14 | secure desktop/反作弊明确失败 | 不保证绕过系统安全边界 | UIPI、secure desktop 与反作弊属于显式 unsupported 边界；adapter 不把注入失败伪装成功，使用稳定 target/input failure 诊断 | verified unsupported boundary | Windows Adapter / G06 |

## Recording, screenshots and assets

| ID | 能力 | 3.0 行为 oracle | 当前事实 | 3.1 决策 | Owner / gate |
| --- | --- | --- | --- | --- | --- |
| REC-01 | simple 键盘+点击录制 | 显式 simple 模式 | simple policy 与 native hook/canonicalizer/asset round-trip 组合证据闭合 | verified | Recording Session / G07 |
| REC-02 | precise 鼠标轨迹录制 | 显式 precise 模式与专用 playback | native precise hook 捕获轨迹/按钮并通过 canonical codec | verified | Recording Session / G08 |
| REC-03 | HUD 状态、暂停、继续、完成 | HUD 与 service 状态同步 | snapshot/event、pending、晚挂载、暂停继续/finalize 生命周期闭合 | verified | Recording Session / G07–G08 |
| REC-04 | finalize/codec/asset round-trip | 真实录制可保存 | native recorder 事件归零后 encode/decode、asset/blob save/reload 成功 | verified | Recording Session / G07–G08 |
| REC-05 | clip playback | 简易与精确回放 | reloaded native clip 与 driver playback 投递 key down/up 并 ReleaseAll | verified | Playback / G07–G08 |
| ASSET-01 | 截图保存模板 | ScreenPicker/模板资产 | native PNG capture + BlobRef journal + same-target template click integration 闭合 | verified | Capture + Asset / G06 |
| ASSET-02 | 搜索、分类、标签、分页、批量 | Asset dock/pager | 共享服务端 query/revision cache、批量管理和 WebView 资源库人工 UX 检查通过 | verified | Asset Query / G09 |
| ASSET-03 | 节点选择模板/clip | TemplatePickerField 打开完整 picker | Inspector 复用搜索式 picker，exact variant 选择不再全量 list/select | verified | Asset Picker / G09 |
| ASSET-04 | 1000 assets 下可发现和可选择 | 搜索式资源面板 | 1000×2 fixture、分页预算、revision invalidation 与搜索选择通过 | verified | Asset Picker / G09 |
| ASSET-05 | stable asset/variant identity + BlobRef | 资源与 variant 绑定 | exact variant BlobRef 持久化与反查/stale 语义通过 | kept / verified | Asset Store / G09 |
| ASSET-06 | 安全清理未引用数据 | 旧维护实际清 subgraph，不是可信 Blob GC | 真实 Application inventory + Asset cleanup 以 Source/Program/Run roots 保护 Blob；orphan 被回收、live workflow Blob 保留 | rebuilt / verified | Asset GC / G09、G12 |

## Node capability continuity

| ID | 能力 | 3.0 行为 oracle | 当前事实 | 3.1 决策 | Owner / gate |
| --- | --- | --- | --- | --- | --- |
| NODE-01 | Stopwatch Start/Read/Stop | 专用节点族 | Start 输出显式时间戳，Read/Stop 仅消费该值；无进程级隐式表 | verified | Runtime State / G10 |
| NODE-02 | WaitStable/WaitChange | 视觉/窗口等待节点 | exact-target 有界帧差、取消、journal 与 executor/admission/provider 集成通过 | verified | Observation / G06 |
| NODE-03 | typed Switch | 动态 Switch | generic equatable Switch 支持最多 8 cases、first-match/default | verified | Control Nodes / G10 |
| NODE-04 | LoadImage/SaveImage | 图像 I/O | managed workspace PNG → BlobRef → PNG exact-byte round-trip，staged durable write | verified | Asset Nodes / G09 |
| NODE-05 | VarLastChange/IncVar | 便利状态节点 | LastChange 与原子 Increment 可从 State 面板直接创建；并发 revision 证据通过 | verified | State / G10 |
| NODE-06 | Mouse calibration 可发现入口 | calibration node/service | 不恢复为 workflow node；Settings input profile、F8 HUD 与 recording metadata 提供机器级校准，Source 不承载宿主参数 | replace / verified product capability | Target Tools / G05 |
| NODE-07 | specialized vision | color scan/track/signature | 用 Capture + FindColorBlobs/AnalyzeColor + typed list/geometry/math 复合替代；旧名称作为搜索别名 | replace / verified | Composition + Catalog / G06、G10 |
| NODE-08 | EventTick | 周期触发 | 删除 ambient background subrunner；Run 内用 Repeat/Wait/Delay，独立周期用 Schedule interval | remove / bounded alternatives | Scheduler / G01 |
| NODE-09 | arbitrary RunProgram | 任意路径启动 | 与安装/authority 模型冲突 | remove; use installed app | Installation / G06 |
| NODE-10 | yt/ambient JS/Expr escape hatch | 宿主内任意脚本和变量表达式 | 与 3.1 隔离/强类型冲突 | remove; isolated Script only | Script conformance |

## Platforms

| ID | 能力 | 3.0 行为 oracle | 当前事实 | 3.1 决策 | Owner / gate |
| --- | --- | --- | --- | --- | --- |
| PLATFORM-01 | Windows 完整支持 | 主平台 | Windows native、UnrealWindow、workspace recovery、WebView 与 production gates 通过 | verified | Slice 34 |
| PLATFORM-02 | Android/ADB emulator | controller、节点和工作流 | exact serial 应用发现、capture/template/input/clip/stop 通过 emulator 的 Source→journal 真实旅程 | verified | Slice 36 / G16 |
| PLATFORM-03 | Browser CDP | 旧底层 controller/client | exact endpoint/page 在受控 Chrome、Edge 完成 Type Text/Capture/journal；operation narrowing 与 identity rotation 通过 | verified | Slice 36 / G17 |
| PLATFORM-04 | macOS 扩展不改 core | 旧版无完整实现 | no-runtime Adapter 独立注册/seal/fail-closed，darwin/arm64 与 linux/amd64 compile proof 未改 core | architecture proof complete | Slice 36 / compile fixture |

## Old evidence anchors

固定 reference worktree：`E:\projects\organizations\yottaapp\yotta-3.0-reference`，commit `8316d590dbc8429b783b99982ff30d15e650c59a`。

主要取证入口：

- `frontend/src/views/ContainerEditorView.vue`
- `frontend/src/components/containers/`
- `frontend/src/components/containers/dock/AssetDockPanel.vue`
- `frontend/src/composables/containerEditor/`
- `frontend/src/views/tools/RecordingHUDView.vue`
- `internal/services/recording/`
- 旧 node/spec/runner、window capture、Android controller 与 input/capture adapter

旧实现仅证明用户行为，不授权恢复 Container、ambient Expr/yt、旧 Pin DTO 或第二套 runtime。

## Update rule

- 每个实现 Slice 只能更新自己负责的 ledger rows。
- `reverify` 只有在对应 golden journey 留下自动化与真实宿主证据后才能变为 `verified`。
- Slice 35 已关闭全部 `defer-decision`；以后新增延期项必须同时记录决策期限、复查证据和不阻塞范围。
- R5 要求所有 P0/P1 行为为 `verified`、`remove` 或有用户明确接受的删除决定；不存在“代码完成、smoke 后补”。
