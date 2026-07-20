# Yotta 3.1 golden journeys

## Evidence contract

黄金旅程是 Stage 完成门禁，不是逐个小改动都重复执行的 micro-test。实现批次只跑必要定向检查；Stage 结束时批量执行相关旅程、聚合测试、`task check` 和触发的 native smoke。

每份旅程证据必须记录：artifact/commit、workspace fixture、目标环境、步骤结果、结构化错误、日志/journal 关联 ID、截图或录像（需要时）以及操作人。页面存在、按钮可点或 mock adapter 成功不算 native 能力证据。

## G01 — 新建、创作、运行与调试

Fixture：clean workspace；基础 pure-data workflow。

1. 新建工作流，自动出现唯一 Run Start。
2. 搜索并拖入 Repeat、Greater Than、Log/可观察节点。
3. 从 Repeat.index 拖出候选，建立可用的 typed 数据链；不手写 JSON。
4. 框选、复制、Delete、撤销、对齐和自动布局。
5. 保存、编译、运行；设置断点并执行暂停、单步、继续。
6. Run timeline 能定位节点，失败显示结构化原因。

通过：无节点误移；无隐藏转换；保存成功不 toast；所有操作复用同一 Source revision 与 scheduler。

## G02 — Windows 捕获即安装

Fixture：普通窗口、管理员窗口、标题含尾随空格的 UnrealWindow。

1. 从未安装状态开始捕获。
2. 捕获期间临时注册全局 F9，完成或取消后释放。
3. 一次确认生成 application identity、target installation 和 explicit consent。
4. exact title 逐字符保存；切换 regex 后验证 RE2。
5. 不重启 Yotta，立即在录制、截图、Inspector 和 Run 中选择同一 slot。

通过：同一 manifest generation 派生入口、authoring、admission、provider 和 policy；取消是 no-op；没有启动快照撕裂。

## G03 — 多窗口与动态标题

Fixture：VS Code 多窗口/动态 tab 标题、QQ 多窗口。

1. 用 exact、regex 分别产生多个候选。
2. UI 显示候选身份与选择策略。
3. `unique` 冲突明确失败；受约束前台/顶层策略确定性选中。
4. 窗口关闭重开后重新解析，不把 HWND 写入 Source。

通过：选择可预测，歧义不会随机命中其他窗口。

## G04 — target generation、删除与 consent

1. 新建、修改 selector、撤销 consent、重新授权、删除 target。
2. 每次 mutation 后同进程立即运行 validation、recording 和 admission。
3. 已运行 Run 保留租约；新 Run 只看新 generation；空闲旧 generation 被回收。
4. 删除 application 时原子处理依赖 target，不留下 unknown slot。

通过：Settings 仅持久化意图；无正常重启提示；无 provider/resource 泄漏。

## G05 — Windows 键鼠输入

1. 通过组合键录入器配置 Press Keys，不编辑 JSON。
2. 执行 Type Text、点击、移动、相对移动、拖拽。
3. 执行 held key/mouse，并在失败、取消和退出时强制释放。
4. 普通窗口和管理员窗口各执行一次。

通过：capability 由 manifest 自动投影；journal 记录 adapter action；错误提供可执行恢复动作。

## G06 — 窗口、截图与模板动作

1. 激活、等待、置前、移动缩放、读取状态、关闭目标窗口。
2. 截取目标窗口模板并保存。
3. 用 Wait Template、Click Template 运行；验证 timeout/failed/error channel。
4. secure desktop 或不支持目标返回明确 unsupported，不伪装 target unavailable。

通过：窗口操作与 input/capture 共享 installation lease 和 exact target resolver。

## G07 — simple recording round-trip

1. 显式选择 simple 模式并开始录制。
2. HUD 晚于 session 打开仍能显示当前状态；暂停/继续有效。
3. 录制按键和鼠标点击，完成并保存 clip。
4. 重新查询资源库，把 clip 插入 workflow 并 playback。

通过：首事件归零、顺序严格有效、release 合法；recorder → canonicalize → codec → asset → reload → playback 全链使用真实事件。

## G08 — precise recording round-trip

1. 显式选择 precise 模式，录制连续鼠标轨迹、拖拽与按键。
2. 中途暂停/继续，暂停时间不进入 playback timeline。
3. 保存、重新加载并回放。

通过：模式不由偶然 mouse move 猜测；轨迹时序、按钮状态和最终坐标在 native smoke 中可信。

## G09 — 1000 项工作流与资产管理

Fixture：至少 1000 workflows、1000 assets、多 variants、失效引用。

1. 搜索、筛选、排序、分页、批量选择/删除。
2. 从节点 Inspector 打开统一 Asset Picker，按名称/标签/类型搜索并预览 variant。
3. 选择 exact variant 后只写稳定 identity/BlobRef，不把全库载入 Inspector。
4. 清理预览列出完整 roots；执行后已引用 Blob 不丢失。

通过：响应预算和查询次数受约束；空列表、翻页、删除引用项均有恢复路径。

## G10 — typed State、转换和便利节点

1. 在大量 Run State 中搜索，拖出读取/写入节点。
2. Repeat.index 连接到数值比较、格式化和日志；候选展示合法转换。
3. 使用 typed Switch、Stopwatch、IncVar/LastChange。
4. 修改 State 类型前预览跨图影响并保持 Source 可编译。

通过：前端候选与 Go compiler 使用相同 connection plan；不存在 ambient variable/Expr fallback。

## G11 — 结构化错误与恢复

覆盖：RPC transport failure、字段校验、target unavailable、consent missing、adapter unsupported、recording finalize、asset missing。

通过：transport decode 后 rethrow；domain action 决定 inline/modal/failure toast；不出现 browser alert、成功 toast、`undefined` 伪成功或二次假错误。

## G12 — 多图与 portability

1. 创建 subgraph、定义 typed interface、插入 GraphCall、调试跨图路径。
2. 添加 annotation/reroute presentation。
3. export canonical Source + exact blobs，在 clean workspace import。
4. 缺 blob、递归或不兼容 contract 给出确定性诊断。

通过：不恢复旧 virtual marker/Container；compiler/program/journal graph path 一致。

## G13 — workspace 启动与损坏恢复

覆盖 clean data、当前开发 workspace、未知 slot、缺失 BlobRef、损坏 JSON。

通过：应用可启动；坏对象被隔离并给出修复/删除动作；不要求用户删除整个 data；不添加未发布 schema 兼容层。

## G14 — AI installation

配置自定义 API URL、模型、credential binding；运行一次 AI 节点并核对 endpoint 只存在于可信安装，Source 不含 credential。

## G15 — launcher 可发现性

从主壳启动悬浮窗，执行一个安全 workflow；退出和重复打开不泄漏窗口或后台资源。

## G16 — Android/ADB

安装 emulator/device profile，选择应用/设备，创作 input/capture/template/clip 工作流并运行；断连、unauthorized、多设备与重连均有确定性诊断。

通过：使用与 Windows 相同的 manifest/admission/journal 路径，平台字段不进入中央 ProfileDraft。

## G17 — Browser CDP

安装 exact endpoint/page profile，在受控 Chrome/Edge 执行已声明 operation；endpoint 变化触发新 generation 与 consent digest，未声明操作 fail closed。

## R2 evidence — 2026-07-18

- G02–G06：`task windows:smoke:automation` 覆盖 exact 尾空格、regex 动态标题、ambiguity、F9、input/held/playback/window/capture；真实 `HTGame.exe / UnrealWindow / 异环··` Run `019f7556-279d-711a-9b98-db9bd616bf94` 成功。
- G07–G08：native hook 事件直接进入 canonicalize → InputClip v3 codec → Blob/Asset → reload → replay。
- G11/G13：corrupt Source 隔离可修复/删除；stale Program 重建；stale consent 只撤授权；clean/corrupt WebView fixture 和当前 workspace 无需删 data。
- G15：真实 Wails WebView 新建安全 workflow、写入 launcher、悬浮窗执行到 success、隐藏后二次打开复用同一 CDP target；人工检查四张 PNG。
- Stage gate：`task check`、`task build`、native smoke、相关 cross-compile、requireAdministrator manifest 与版本资源通过。Yotta.exe SHA-256 `7652263517690B0A527DAE2F40810E456FB97AF60BA09B79A75E49536FAB136D`。
- unsupported：不承诺 secure desktop、反作弊或更高完整性注入；旧未发布扁平 target schema 被拒绝，不增加迁移层。

## R3 evidence — 2026-07-18

- G01：Wails WebView clean/corrupt fixture 完成 Start、目录搜索、插入、连接、多选 Delete、对齐/自动布局、编译、debug 与时间线；145-node catalog，最终 canvas 4 nodes，AI review 成功。
- G06：WaitStable/WaitChange 通过真实 Executor → admission → provider exact-target capture 集成，验证两帧观察、取消和 journal；Windows 窗口 lease/resolver 继续采用 R2 native 证据。
- G09：workflow service 真实创建并查询 1000 Sources；Asset Query 使用 1000×2 fixture；State authoring 使用 1000 states fixture。查询均有有界分页/预算，Inspector 复用搜索式 Asset Picker。
- G10：原子 State Increment/LastChange、typed generic Switch、显式 Stopwatch、Repeat.index 候选与 conversion/compiler parity 通过；100 个并发 Increment 得到 exact value/revision 100。
- G11–G12：WebView 完成结构化失败恢复、同 scheduler debug、typed subgraph/GraphCall 与 AI review；workspace-safe PNG source → BlobRef → saved PNG 通过 exact-byte Executor round-trip。
- Stage gate：`task check` 全绿（Go、vet/staticcheck、coverage、44 个前端文件 185 tests、Wails contract、production frontend 与 bundle budgets）；WebView smoke 目录 `.task/workflow-editor-smoke/20260718-223440/`，`workflow-editor.png`、`assets.png`、`workflows.png` 已人工检查。
- R3 不要求 UAC：本阶段只复用 R2 的 Windows native installation/capture/input 证据；下一次 UAC 验证仅在平台或最终 native matrix 触发时执行。

## R4 evidence — 2026-07-18

- G16：`bilibili_api35 / emulator-5580` 从 exact serial/package 安装进入 Source → Compiler → Admission → installed provider → journal，activate、capture、template click、drag、InputClip playback、stop-app 全部成功；断连、unauthorized、多设备和 reconnect 有定向诊断。
- G16 可靠性：真实旅程发现 effect 子进程缺少 Adapter-owned deadline；所有 ADB command 现在默认 10 秒上限并继承更短上游取消，阻塞 runner 回归证明 Run 不会无限 RUNNING，输入副作用不自动重试。
- G17：受控 Chrome 与 Edge 均完成 exact page discovery、Type Text、Capture、页面副作用与 journal；未声明窗口 operation fail closed，endpoint/page 漂移旋转 generation 与 consent digest。
- macOS seam：no-runtime Adapter 独占 profile/descriptor/manifest 并显式 unavailable；`internal/automation/installed`、`internal/appbootstrap` 的 darwin/arm64 与 linux/amd64 cross-compile 通过，未改 Source、通用节点、compiler 或 scheduler。
- Stage gate：`task check`（282.4s）、`task build`（36s）、Android/Chrome/Edge 真实 smoke、contracts、bindings、production manifest 全绿；Yotta.exe SHA-256 `F7B996866CD82A79493BDF8139274910B8C8E43AD751E9DDEC28677FE740C3D2`。
- 清理：controlled browser profile/process 与 emulator 精确回收，`adb devices -l` 为空；本阶段无需用户以 UAC 重开 Yotta。

## R5 evidence — 2026-07-19

- G01–G17：`task check` 0（176.4s）；`task build` 0（20s）。
- G12/G14：独立 stores 导入/导出 exact Blob；custom AI endpoint 经 Source→Compiler→Admission→provider→journal，核对 path/auth 且 Source 无 secret。
- G09/G12：真实 inventory + cleanup 只回收 orphan，保留 live Source Blob。
- 平台：Darwin/Linux compile；Windows Process/Wasm/UAC manifest；Chrome/Edge；Android emulator-5554 旅程 11.6s 后清理。
- Windows native：管理员子进程完成 input/recording/F9/window；夹具在 `DefWindowProc` 后发布 down observation。
- Frozen 3.1.0 candidate exact manifest/digest、workers/plugins/CLI 与管理员 startup 通过。
- WebView 68.4s：145-node catalog、4-node canvas、损坏 Source、launcher、AI review；人工查看四张 PNG。
- `Yotta.exe` SHA-256 `F7B996866CD82A79493BDF8139274910B8C8E43AD751E9DDEC28677FE740C3D2`；仅待 1920/UAC 用户验收。

## Stage mapping

| Stage | Required journeys |
| --- | --- |
| R1 perimeter contracts | G04、G07、G08、G09、G11 的 contract/integration 部分 |
| R2 Windows | G02–G08、G11、G13、G15 |
| R3 editor/capability | G01、G06、G09–G12 |
| R4 platform | G04、G09、G11、G16、G17；另含 macOS compile proof |
| R5 release | G01–G17 的适用完整矩阵 |
