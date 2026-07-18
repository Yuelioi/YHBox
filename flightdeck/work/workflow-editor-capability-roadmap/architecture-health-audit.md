# Yotta 3.1 架构健康与产品连续性审计

## 审计结论

当前 3.1 **不具备发布条件**，此前“major upgrade 完成”和“发布阈值已满足”的结论无效。

这不是一个需要继续逐个修 bug 的局部质量问题。3.1 的执行内核方向总体健康，但产品外围在一次大规模替换中失去了多个深模块边界和端到端能力闭环：自动化安装、运行时激活、录制、资产选择、错误传播、平台 profile 和阶段验收互相割裂。继续在现有边界上追加条件分支，会让同类问题反复出现。

建议采用 **保留内核、重建边界、以旧版行为作 oracle 的纵向恢复**：

- 不回滚到旧 Container/双运行时。
- 不继续按单个报错打补丁。
- 固定 `8316d590dbc8429b783b99982ff30d15e650c59a` 为 3.0 行为基线，用独立 worktree 观察用户流程，不迁移旧内部架构。
- 保留 3.1 Source → Compiler → Program → Executor、严格类型、Run journal 和 capability admission 核心。
- 重建外围深模块，每个阶段交付一组完整用户旅程，阶段末统一测试、构建和真机验收。

## 范围与证据

- 指定旧基线：`8316d590dbc8429b783b99982ff30d15e650c59a`，2026-07-12 23:41:59，`feat(recording): add asset lifecycle controls`。
- 基线至当前 HEAD：245 个提交，1607 个文件变化，新增 134,578 行、删除 118,893 行。
- 审计对象：当前与旧版的 workflow editor、node/runtime、recording、asset/template、automation installation、admission、settings、frontend RPC 和 Flightdeck 验收记录。
- 本轮没有修改业务代码、运行构建、杀进程或制造用户数据。
- 主流系统对照不重新做泛化搜索；使用仓库已有的一手资料研究：
  - [`docs/research/node-system-mainstream-practices.md`](../../../docs/research/node-system-mainstream-practices.md)
  - [`docs/research/visual-type-system-authoring.md`](../../../docs/research/visual-type-system-authoring.md)

旧版代码只作为行为与能力证据，不把旧实现质量当作目标架构标准。

## 总体健康判断

| 区域 | 判断 | 处理 |
| --- | --- | --- |
| Source → Compiler → Program → Executor 唯一路径 | 健康方向 | 保留并加纵向 conformance |
| exec/data/error 通道、严格类型和显式转换 | 健康方向 | 保留；继续由后端契约投影 |
| immutable Program、内容摘要、durable Run journal、同 scheduler 调试 | 健康方向 | 保留 |
| controller 与 adapter registry 概念 | 有价值但 seam 太浅 | 深化为 adapter-owned profile/schema/editor/runtime |
| capability 声明与 Host Profile 投影 | 不健康 | 删除手工二次映射，单一 manifest 派生全部视图 |
| target 设置、激活、authoring、admission、provider 生命周期 | 不健康 | 合并成 generation-owned 深模块 |
| recording 状态机、事件规范化、保存和 HUD | 不健康 | 作为一个完整子系统重建并端到端验收 |
| 资产查询、节点选择器与 BlobRef 绑定 | 不健康 | 统一查询/选择契约，禁止全量下拉 |
| frontend RPC/错误边界 | 不健康 | transport 规范化后 rethrow，使用场景决定 inline/toast |
| 页面与 editor session | 过浅且体积过大 | 按职责提取深模块，不做纯文件切割 |
| 阶段验收 | 不可信 | 撤销发布完成结论，改用真实旅程门禁 |

结论不是“3.1 全部推倒”。真正值得保留的是新的执行与契约内核；需要重构的是它与用户、平台和资源之间的边界。

## 已确认的架构失效模式

### 1. Capability 存在两个事实源

节点契约分别要求通用 input、desktop input、key input 等 capability；adapter descriptor 又声明 operations/resource kinds；production composition 再通过 `builtinHostProfile` 手工把后者翻译回 capability。

真实 Press Keys 工作流因此出现过：provider 存在、目标存在、节点可编译，但 Host Profile 漏投影 `automation/key-input`，最终误报 `admission.target_unavailable`。添加一个 `switch` 能修当前节点，却没有修复第二事实源。

需要改为一个可执行、版本化的 installation manifest，由它一次派生：

- authoring target descriptor；
- admission Host Profile；
- provider registry；
- policy/consent 摘要；
- Settings 可编辑 schema 与健康诊断。

任何 operation 新增后，不允许再修改 composition root 的手工映射。

### 2. 目标运行时生命周期泄漏到 composition root

当前热更新需要同时替换 Host Profile、Policy、providers 和 authoring resolver。旧 installation generation 被保留到进程退出，Settings 还参与 prepare/commit/abort 激活协议。

这说明 Interface 太宽，Settings、Application 和 runtime 都知道同一生命周期细节。它已经产生“设置已保存、录制能看到、Admission 看不到”的启动快照撕裂，也会在重复修改目标后积累旧 native provider。

需要一个 `AutomationTargetRuntime` 深模块独占：seal、构建、原子发布、run lease、旧 generation 回收、shutdown。Settings 只保存用户意图；录制、资产、authoring 和执行只持有稳定 handle。

### 3. 平台 Adapter 只抽象了 runtime，没有抽象 profile 全链

当前 `ProfileDraft` 集中包含 Windows executable/window/input/capture、Android ADB、Browser CDP 字段；seal、settings schema 和前端表单都按 adapter kind 分支。增加 macOS 仍需同时修改 Go 中央结构、验证、Settings API、TypeScript 类型和巨型页面。

因此“未来 macOS 只新增 Adapter”的既有结论不成立。

目标 profile 应成为 adapter-owned、版本化的 discriminated payload，例如：

```text
TargetInstallation {
  slot
  targetKind
  adapterKind
  profileVersion
  profilePayload
}
```

adapter 自己提供 schema、validator/sealer、authoring descriptor、runtime factory、health check 和 profile editor descriptor。Windows application identity、macOS bundle identity、Android package/device、Browser endpoint/page 不应继续塞入同一中央 draft。

### 4. Recording 没有守住自己的持久化不变量

`inputclip` codec 要求第一个事件 `TUs == 0` 且 `(TUs, Seq)` 严格有序；真实 recorder 直接保存相对系统时间，没有在 finalize 前把第一事件归零。手写 fixture 都已经满足 invariant，所以单元测试通过，真实 904 事件录制仍报 `input clip event 0 ordering is invalid`。

同时：

- 页面先启动录制再打开 HUD；HUD 只订阅未来事件，不在挂载时读取当前状态，因此错过已发出的 recording 状态并永远停在“准备中”。
- 旧版明确提供 `simple | precise` 模式；当前用“出现任意 mouse move 就判 trajectory”的启发式替代。正常点击前的轻微移动也会改变产物语义。
- 旧版 precise playback 有独立 player、QPC timing 和 Windows hybrid backend；当前替换后缺少可信的等价真机证据。

修复单位必须是完整 Recording Session：状态快照 + 事件流、规范化、暂停时钟、simple/precise 策略、finalize、codec round-trip、asset save、HUD reconcile、playback。不能再分散到页面、store、service 和 codec 各自“测试通过”。

### 5. RPC wrapper 吞掉错误并制造第二个假错误

前端通用 `invoke()` 捕获 RPC 异常、自动 toast，然后返回 `undefined`。调用方随即把 `undefined` 当业务结果校验，再抛出 `recording.finalize: invalid result`。

这正好解释用户同时看到原始保存失败和虚假的 finalize 失败。它也使调用方的 `try/catch` 失效，无法根据错误码提供原地恢复动作。

Transport 层只能完成：错误解码、结构化、关联 operation/run id，然后 rethrow。页面或领域 action 决定 inline message、字段错误、可重试动作或 failure toast。成功的保存、安装等原地动作不 toast。

### 6. 资产库和节点选择器是两套产品路径

当前资源库已经有查询、分页和批量能力；节点 Inspector 却把 `asset × variants` 全量展开为普通下拉，并直接绑定 immutable BlobRef。真实数据里模板已经存在，资源库能看到，但节点下拉为空，说明节点选择路径没有共享资源库的查询/缓存生命周期。

即使空列表 bug 被修复，1000 个资源的普通下拉仍然不可用。旧版已有 `TemplatePickerField` → 完整 Asset Panel 的搜索、缩略图、过滤、分页、详情和 picker mode；3.1 删除了这条交互闭环。

需要统一 `AssetPickerQuery`：服务器分页、搜索、类型/标签过滤、缩略图预算、最近使用、variant 明细和稳定选择身份。节点 Inspector 只打开 picker 并接收 exact BlobRef，不持有全量资源数组。

### 7. 巨型模块暴露出低 locality

当前代表性文件：

- `WorkflowEditorView.vue` 约 2646 行；
- `EditorSession.ts` 约 2299 行；
- `SettingsAutomation.vue` 约 1301 行；
- `AssetsView.vue` 约 1085 行；
- `application.go` 约 1239 行；
- `compiler.go` 约 853 行。

行数不是独立罪证，但结合上述跨文件生命周期和重复事实源，说明修改一个用户流程必须理解太多内部细节。重构目标不是“拆小文件”，而是形成少量深 Interface：Target Runtime、Recording Session、Asset Picker、Typed RPC Boundary、Editor Authoring Commands。

### 8. 阶段验收验证了组件存在，没有验证产品闭环

本 topic 早已定义能力必须覆盖“可见入口—管理流程—创作绑定—运行闭环”四层，但后续多个 Slice 依赖源码存在、WebView 可见、手写 fixture、`task check` 和延期真机 smoke 宣告完成：

- Slice 7 声称录制完成，真实 recorder → finalize → codec 没有集成门禁。
- Slice 13 声称 macOS 只需新 Adapter，但 profile/schema/UI 仍中央分支。
- Slice 17 声称资产规模完成，节点 picker 仍是全量下拉。
- Slice 20 实现与状态记录互相冲突，真实宿主 smoke 一直未闭合。
- Slice 26 承诺空闲 generation 回收，当前仍保留到 shutdown。

批量验收原则本身没有错；错的是阶段边界。以后不为每个小改动反复跑全量门禁，但 Stage 不能在其关键真实旅程未通过时完成。

## 旧版能力连续性

静态节点数量不能代表产品能力。旧版大约 93 个可识别 node kinds，当前约 124 个 URI；当前增加了大量类型、数学、文本与转换节点，但仍丢失或改变了重要自动化能力和创作流程。

### 保留或增强

- Run start、Branch、Repeat、Delay、基础键鼠、Play Clip、模板等待/点击、HTTP/AI、状态读写等已有新契约对应物。
- 图选择、Delete、clipboard、对齐、布局、节点搜索、连线候选、运行轨迹和 debugger 已有新实现。
- 强类型、显式转换、typed State、多图 Source 和 capability admission 比旧版更适合作为长期内核。

### 已确认缺失或没有等价闭环

| 旧能力 | 当前判断 | 3.1 决策 |
| --- | --- | --- |
| Wait/Get/Close/BringForeground/MoveResize/WindowState 等窗口操作 | 除 Activate 外多项缺失 | 恢复为平台 capability 下的节点族 |
| KeyHoldStart/Stop、MouseHoldStart/Stop | 缺失 | 恢复；用 run lease 保证异常释放 |
| Stopwatch Start/Read/Stop | 缺失 | 恢复纯运行状态节点 |
| WaitStable/WaitChange | 未找到等价闭环 | 结合视觉/窗口观察器重做 |
| 动态 Switch | Branch/Select 不能完全替代控制流 Switch | 设计 typed Switch |
| LoadImage/SaveImage | 通用文件节点不能替代图像资产语义 | 通过 BlobRef/asset capability 恢复 |
| DualColorBarTrack、ROIColorScan、FindColorSignature | 泛用视觉节点不等于行为等价 | 按真实工作流决定专用复合节点 |
| VarLastChange、IncVar | 可组合但创作成本回归 | 作为 typed 便利节点或原子命令恢复 |
| MouseCalibration node | 只剩服务/UI 构件 | 恢复可发现入口与 target 绑定 |
| EventTick | 被刻意移除 | 不直接恢复；先证明 schedule/trigger 用户替代闭环 |
| RunProgram | 安全原因改为 installed application | 不恢复任意路径执行；补齐安装/启动体验 |

### 被删除但需要逐项确认用户等价物的创作能力

旧前端删除的组件包括 CommandPalette、DebugPanel、ProblemsBar、KeyCapture、Recording Save/Cleanup、ClipTimeline、TemplatePickerField、Asset dock panels、NodeSearchModal、SnapGuide、context menus、dynamic inputs/outputs 等。部分能力已经以新组件恢复，不能仅凭文件删除判定缺失；下一阶段以用户旅程逐项验收，而不是按组件名机械搬回。

## 已有主流实践对本次问题的验证

仓库现有研究已经足够，不需要再做一轮泛化调研：

- Unreal/Blockly/Unity 的共同启示是 contract/schema 驱动创作投影、上下文候选和可见转换，而不是前端猜类型。3.1 类型内核方向正确。
- Node-RED 的节点与错误实践强调 adapter 边界规范化异步错误；这与当前 RPC 吞错相反。
- n8n/JSON Schema 的版本实践支持 versioned contract/profile；当前中央 `ProfileDraft` 不满足 adapter-owned versioning。
- Unity Blackboard 和 Node-RED typed input 都把搜索与值来源选择作为基础控件；这证明普通全量下拉不适合 State/Asset/Target 等规模化引用。
- capability 声明、授权、调度 eligibility 必须分层且从同一契约派生；当前 capability 二次投影违反已有研究结论。

因此当前主要问题不是“还没找到正确理论”，而是实现和验收没有遵守仓库已经接受的设计原则。

## 恢复方法比较

| 方法 | 判断 | 原因 |
| --- | --- | --- |
| 回滚到 3.0 后重新升级 | 不采用 | 会丢掉有价值的唯一执行内核、严格类型和安全边界，再次产生双栈迁移 |
| 在当前分支继续逐 bug 修补 | 不采用 | 重复事实源和浅 seam 不消失，同类故障会继续出现 |
| 全部重写 3.1 | 不采用 | 范围过大，也会重写当前健康内核 |
| 保留内核、重建外围深模块、纵向恢复 | 采用 | 既保住新架构收益，又以用户旅程控制风险 |

## 发布前恢复路线

详细执行顺序见 [`slices/27-architecture-recovery.md`](slices/27-architecture-recovery.md)。高层阶段如下：

1. **R0 重置事实与门禁**：冻结新增 feature；建立旧/新双 worktree 行为 oracle、能力 ledger 和黄金旅程；撤销不可信完成状态。
2. **R1 重建基础边界**：Typed RPC Error、Automation Target Runtime generation、单一 Installation Manifest、Recording Session、Asset Picker Query。
3. **R2 Windows 核心纵向闭环**：安装/捕获/授权无需重启；按键、鼠标、截图、模板、simple/precise 录制与 playback 在普通/UAC/多窗口目标通过。
4. **R3 编辑器与资源创作闭环**：搜索式节点/State/Asset/Target picker，模板和 clip 从创作到运行；恢复确认过的窗口、held input、stopwatch 等 P0/P1 节点。
5. **R4 Android 与平台 seam 证明**：用 Android adapter 接入同一 manifest/profile/runtime/picker，不修改核心；用一个最小 macOS profile contract compile proof 验证 seam，而非承诺 runtime 完成。
6. **R5 发布验收**：阶段级批量 `task check`/build；Windows 真机、ADB emulator、clean/dirty workspace、1000 资产、退出重启和失败恢复矩阵。

## 新的完成定义

一个能力只有以下证据齐全才能标记完成：

1. 用户从可见入口可发现、取消、失败恢复；
2. 管理层支持必要的创建、编辑、删除、搜索和规模；
3. editor 保存 exact typed reference，不靠 title/path/临时 handle 猜测；
4. compiler、admission、provider/runtime 和 journal 使用同一契约；
5. 至少一个从真实输入到真实副作用/持久化 round-trip 的自动化集成；
6. 属于宿主能力时，在对应 Stage 末通过真实 Windows/ADB smoke；
7. 只有上述旅程通过后，才在阶段末统一跑全量门禁并更新状态。

源码存在、单个 unit test、页面截图、build 成功或“后续再做真机 smoke”都不能单独证明产品能力完成。
