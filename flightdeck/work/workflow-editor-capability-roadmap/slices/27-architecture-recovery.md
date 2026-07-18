---
slice: "27"
title: 3.1 架构恢复与产品纵向闭环
status: in_progress
---

# Slice 27：3.1 架构恢复与产品纵向闭环

## Outcome / Question

停止围绕单个报错继续打补丁。保留 3.1 健康执行内核，以 3.0 指定提交为行为 oracle，重建自动化目标、录制、资产选择和错误边界四个外围深模块，并按真实用户旅程恢复 Windows、Android 与编辑器产品连续性。

架构审计见 [`../architecture-health-audit.md`](../architecture-health-audit.md)。

## Execution routing

R0–R5 已按独立 outcome 拆为可执行 Slices；本文件保留为恢复 umbrella 和设计理由，不再直接承载下一个实现动作：

- [Slice 29](29-fact-baseline.md)：事实基线、capability ledger、黄金旅程与历史 Knowledge 退役清单。
- [Slice 30](30-typed-rpc-error-boundary.md)：Typed RPC 错误边界。
- [Slice 31](31-installation-manifest-target-runtime.md)：Installation Manifest 与 Target Runtime。
- [Slice 32](32-recording-session.md)：Recording Session。
- [Slice 33](33-asset-picker-query.md)：Asset Picker Query。
- [Slice 34](34-windows-native-closure.md)：Windows native 纵向闭环。
- [Slice 35](35-editor-capability-recovery.md)：编辑器与节点能力恢复。
- [Slice 36](36-platform-adapter-continuity.md)：Android/Browser/platform seam。
- [Slice 37](37-release-gate-knowledge-retirement.md)：发布门禁与 3.0 Knowledge 退役。

## Blocked by

无。Slice 28 已完成旧 Knowledge/docs 复查；R0 可直接开始。

## Verification

每个 Recovery Stage 使用该 Stage 定义的黄金旅程与聚合 gate；完整 `task check`/production build/真实宿主矩阵只在相应阶段末统一执行。详细证据写入 capability ledger，不以源码存在、数量或页面截图替代。

## Out of scope

- 不回滚或复活旧 Container/第二 runtime。
- 不迁移未发布数据，不为旧内部 schema 增加兼容层；旧版本只用于行为对照。
- 不按缺一个节点就加一个特判的方式恢复能力。
- 不在此 Slice 中承诺 macOS runtime；只证明新增平台不需要修改核心契约。
- 不把单测、页面可见或 production build 当成宿主能力完成。

## Execution rule

- 一个 Recovery Stage 包含多个相邻实现批次；批次内只运行继续开发所需的定向检查。
- Stage 完成时统一运行相关聚合测试、`task check`、production build 和该 Stage 的真实宿主 smoke。
- Stage 关键旅程未通过时不得标 completed，也不得以“发布后补齐”延期。
- 每个 Stage 完成并形成可回滚边界后再 commit；不为每个小修复单独验收、单独 commit。
- 不清理、不覆盖用户当前 dirty worktree；开始实现前先盘点并划清现有改动归属。

## Stage R0：事实重置与黄金旅程

### Deliverables

- 创建只读 3.0 reference worktree，固定到 `8316d590dbc8429b783b99982ff30d15e650c59a`。
- 建立 capability ledger，每项记录 old behavior、3.1 decision、四层闭环、owner module 和验收证据。
- 建立至少以下黄金旅程：
  - 新建工作流 → Start → 搜索/拖入节点 → 连线 → Delete → 保存 → Run/debug；
  - 捕获窗口 → 自动安装应用/目标 → 精确或 regex 匹配 → 授权 → 同进程运行；
  - simple 键鼠录制 → 保存 clip → 插入 workflow → playback；
  - precise 轨迹录制 → 暂停/继续 → 保存 → playback；
  - 截图模板 → 资源库可见 → 搜索式 picker 绑定 → Click/Wait Template；
  - Android 安装 → 设备选择 → 节点创作 → emulator run；
  - 错误、取消、目标删除、重启后的恢复行为。
- 将 Slices 7、8、9、13、15、17、19、20、26 的完成声明映射到 ledger；没有纵向证据的状态视为待复验。

### Gate

只做文档与可重复 fixture/旅程定义，不改产品行为；用户确认 capability decisions 后进入 R1。

## Stage R1：外围深模块重建

### R1.1 Typed RPC Error Boundary

- backend error envelope 有稳定 code、category、message、details、operation/run id 和 retryability。
- transport 只 decode + rethrow，不 toast、不返回伪 `undefined` 成功值。
- domain action 决定 inline、字段错误、恢复动作或 failure toast。
- 删除保存成功等无意义 toast；禁止 browser alert。

### R1.2 Installation Manifest 与 Automation Target Runtime

- adapter installation 生成唯一、版本化 manifest；authoring、admission、providers、policy digest 和 health 从同一 manifest 投影。
- 删除 `builtinHostProfile` 一类手工 capability 二次映射。
- runtime 独占 prepare/publish/lease/reclaim/shutdown；Settings 只持久化意图。
- 新 generation 原子切换；旧 Run 持有 lease，空闲 generation 可回收。
- 捕获一次完成应用身份、target profile 和 explicit consent，不要求正常重启。

### R1.3 Recording Session

- 统一状态 snapshot + event stream，HUD mount 必须 reconcile 当前状态。
- recorder 产物在 finalize 前 canonicalize：首事件归零、严格排序、pause 时间处理、合法按键释放。
- simple/precise 是显式用户策略，不由 incidental mouse move 猜测。
- finalize → codec → asset → reload → playback round-trip 使用真实 recorder events 测试。

### R1.4 Asset Picker Query

- 统一资源库和节点 Inspector 的分页 query、搜索、filter、thumbnail budget 和 exact variant selection。
- picker 返回稳定 asset/variant identity 与 BlobRef；Inspector 不加载全库，不使用普通全量下拉。
- 至少以 1000 assets/多 variants fixture 验证性能和可发现性。

### R1 gate

- 四个深模块 contract/conformance 测试通过。
- 旧 facts 的手工 projection 和 RPC 自动 toast 路径被禁止性测试覆盖。
- 运行一次 Stage 聚合测试；尚不宣称 Windows 产品闭环完成。

## Stage R2：Windows 核心纵向闭环

### Deliverables

- desktop application/window target 一行一职责布局；F9 临时全局注册、捕获后释放。
- exact title 原样保存，regex 显式选择；多窗口有 deterministic selection，不把 HWND 写进 workflow。
- application/target capture、edit、delete、consent 变更后同进程生效。
- Press Keys、Type Text、click/move/drag、screenshot、template wait/click 和 clip playback 真实生效。
- 普通窗口、管理员窗口、多窗口/动态标题窗口分别验证；secure desktop 与反作弊边界明确报 unsupported。
- 运行失败显示原始结构化原因与可执行恢复动作，不出现二次假错误。

### R2 gate

- 用户提供过的异环、QQ/普通窗口和真实 workspace 旅程通过。
- Windows production exe 从 clean data 和现有 workspace 启动并完成黄金旅程。
- 阶段末统一 `task check`、production build、Windows native smoke；通过后单次 commit。

## Stage R3：编辑器、资源与节点能力恢复

### Deliverables

- State、Node、Asset、Target picker 统一搜索交互；从 output/panel 拖出读取、写入、转换或上下文候选。
- Start、新工作流引导、Delete、layout/alignment、debug、端口提示和连线不误移形成一条验收旅程。
- 先恢复已确认 P0/P1：窗口生命周期、held key/mouse、stopwatch、WaitStable/WaitChange、typed Switch、image asset I/O。
- specialized vision、VarLastChange/IncVar、mouse calibration 按旧真实 workflow 使用证据决定专用节点或复合 authoring command。
- EventTick 不机械搬回；必须先给出 schedule/trigger 的可发现等价旅程。

### R3 gate

- capability ledger 中每个恢复项都有四层闭环证据。
- 1000 assets、较大 node catalog 和大量 Run State 下的搜索/选择可用。
- 阶段末统一 editor/integration gate 与人工 UX 验收；通过后单次 commit。

## Stage R4：Android 与平台 seam 证明

### Deliverables

- Android application/device/target profile 由 adapter 自己定义版本、schema、seal、health 和 editor descriptor。
- ADB input、capture、template/clip capability 使用与 Windows 相同的 manifest/admission/journal 路径。
- 增加一个不提供 runtime 的最小 macOS profile/descriptor compile proof；若必须修改 workflow core、generic node、compiler、scheduler 或 central ProfileDraft，则 seam 验收失败。

### R4 gate

- ADB emulator 完成安装、创作、运行、录制/资源适用能力和错误恢复矩阵。
- Windows 回归不变；cross-platform core/GUI compile gate 通过；通过后单次 commit。

## Stage R5：3.1 发布门禁

- `task check` 与 production build 只在本阶段统一跑完整门禁。
- Windows native：普通/UAC、多窗口、动态 title、重启、取消、删除/重建 target。
- Android：至少一个受控 emulator/device matrix。
- 数据：clean workspace、当前 workspace、损坏/缺失引用的可恢复诊断；不做旧 schema 兼容。
- 规模：1000 workflows/assets/states 的搜索、分页和 picker budget。
- 生命周期：连续编辑/替换 target 后 generation 资源回收；退出不泄漏、不挂死。
- 发布文档只引用真实证据；未通过项不能以 post-3.1 代替。

## Completion criterion

- 审计列出的 P0/P1 架构失效模式已经从边界上消除，而不是添加局部特判。
- Windows 和 Android 关键黄金旅程通过，并留下自动化与真实宿主证据。
- capability ledger 没有“组件存在但产品闭环缺失”的完成项。
- `upgrade-plan.md` 的发布阈值只能在 R5 后重新声明满足。

## Result

In progress。R0–R4 已完成；R5 自动矩阵、frozen candidate 与 3.0 Knowledge 退役已完成。仅等待最终 UAC 用户接受验收。
