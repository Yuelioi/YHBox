# 3.0→3.1 Knowledge / docs 复查上下文

## Scope and provenance

- 旧行为基线：`8316d590dbc8429b783b99982ff30d15e650c59a`。
- 当前实现依据：2026-07-18 working tree；其中包含此前未提交的自动化 target/runtime 修补，不能把 dirty state 当发布证据。
- 复查范围：`flightdeck/knowledge`、`docs/architecture`、`docs/research`、`docs/wayfinder/node-system-3.1`，以及它们指向的当前源码入口。
- 本文件记录 topic-local 差异与文档治理；稳定约定已就地升级到 Knowledge/architecture docs。

当 `internal/nodecontract`、`internal/nodes`、`internal/noderuntime`、`internal/automation/installed`、recording/asset service 或 Windows privilege 决策改变时，需要重做对应行。

## 如何读取这些资料

| 资料 | 当前用途 | 权威性 |
| --- | --- | --- |
| `docs/architecture/*`、ADR | 现行架构入口与接受的不变量 | 规范/当前说明；已显式标出 transitional gap |
| `flightdeck/knowledge` 未标“历史 3.0”的条目 | 可复用实施约定和已验证 trap | 稳定知识，仍需遵守 `recheck_when` |
| `docs/research/*` | 主流系统证据、方案备选和设计理由 | 非规范研究，不直接证明当前实现完成 |
| `docs/wayfinder/node-system-3.1/*` | 3.1 内核设计决策的 provenance | 历史决策输入；最终状态看 architecture/code |
| Knowledge 中标“历史 3.0”的条目 | 旧行为、旧故障和产品连续性 oracle | 只在固定旧 worktree/归档取证时读 |
| archived Topics | 当时任务过程与证据 | 不自动升级为现行约定 |

## 有价值的旧版行为证据

旧知识中仍值得纳入 3.1 恢复的不是 Container 内部实现，而是这些用户语义和事故教训：

- 录制有显式 simple/precise 模式、暂停/继续、全局结束热键、保存后可回放；激活、settle、held input cleanup 属于 provider 原子语义。
- 模板/clip 资产使用稳定管理身份和 immutable content binding；旧 picker 已支持搜索、缩略图、分页、详情和 picker mode。
- 窗口 selector 需要 exact 原文与显式 regex；多窗口不能随机取 HWND。
- Wait/Get/Close/BringForeground/MoveResize/WindowState、held key/mouse、stopwatch、WaitStable/WaitChange 等曾是实际工作流能力。
- 子图 create→update 空壳、跨 Store 写盘、全局“当前容器”和异步 layout 回写等旧事故，共同证明复合 authoring 必须单 revision 原子提交，异步结果必须绑定 source/revision/graph context。
- 旧节点框架多条假绿事故证明：全集规则要遍历语义 Catalog，adapter 必须单独做 conformance，Stage 必须验证真实纵向旅程。
- 旧 yt/Expr/ambient VarStore 虽然好用，但会形成第二 authoring/runtime authority；3.1 不直接恢复，交互价值由 typed patch、context candidate 和 typed State 提供。

## 新旧架构差异

| 3.0 概念 | 3.1 接受替代 | 当前状态 |
| --- | --- | --- |
| `nodepkg.Spec` + init Registry + kind dispatch | versioned Node Contract + sealed Catalog + implementation lock | 内核已成立 |
| Runnable/RegionRunner/Evaluator | ExecutionSpec + host instruction + installed adapter | 已成立 |
| Container graph/runtime/vars | Workflow Source → Compiler → Program → Executor + typed Run State | 已成立；外围旅程待复验 |
| exec DataField + global held-output cache | Program data plan + activation-scoped push output | 已成立 |
| local/global/auto VarStore、GetVar/SetVar | typed State declaration + StateRead/Write/Metadata | 已成立，创作体验需要继续验证 |
| goja Script 节点函数、yt console、Expr | one-shot isolated JSON Script；typed authoring patch；显式节点/转换 | 安全边界已成立，旧便捷性不机械恢复 |
| Win32 target 节点/临时 HWND | installation slot + application identity + exact/regex selector | 部分成立；profile/lifecycle seam 需重构 |
| 节点/target capability 多处分支 | 单一 installation manifest 派生 projection/admission/providers | 目标设计；当前仍有二次投影 |
| 启动时安装快照 | atomic generation + Run lease + idle reclaim | 目标设计；当前过渡实现未闭合 |
| 旧资源 picker/dock | paged AssetPickerQuery → exact BlobRef | 管理页部分成立，节点 picker 未闭合 |
| 旧录制 player/session | Recording Session → canonical inputclip → asset → playback | contract/组件存在，真实 round-trip 失败过 |
| 旧 debug runtime/region step | 同一 Executor scheduler + DebugController | 内核已成立 |
| Android/Browser 专用链 | adapter-owned profile/manifest/runtime | runtime registry 有基础，完整 profile/UI seam 未成立 |

## 本次纠正的高风险漂移

- “macOS 只新增 Adapter”改为：runtime seam 有基础，但中央 `ProfileDraft`/Settings/UI 仍需重构。
- “input 只接受唯一 exact title/class”改为：exact 或显式 regex + deterministic selection。
- “Host Profile 只能启动时冻结”改为：sealed installation generation 原子发布，运行中不能 ambient 热插入。
- “Windows 主进程不提权”改为当前产品契约 `requireAdministrator`，同时保留 capability/guest isolation。
- “nodes31/nodes31runtime 是现行包”改为稳定 `internal/nodes` / `internal/noderuntime`。
- “RPC/Wails/测试/节点数量是长期基线”改为：数量只作观测，tracked contract 与 Task/CI gate 才是权威。
- “资源库分页完成等于节点资产选择完成”改为：管理和创作 picker 必须共用 query/identity。
- “codec/store/HUD 分别测试通过等于录制完成”改为完整 recorder→playback round-trip。

## Knowledge 治理结果

### 已升级为当前 3.1 规则

- feature continuity、multiplatform boundary、installed input、AI installation、application lifecycle、asset subsystem。
- node architecture、add-node、reference、contract style、data flow、typed State、isolated Script、error/validation、debug/recording/conformance traps。
- build 基线去除易漂移的测试/RPC/bundle硬编码数量。

### 保留但收窄为历史 3.0

- Container durability、old PinLiteral/Geometry、yt console、Container LogMerger、Normalize/MCP、Expr/Ctx/held-output/PinValue。
- 旧 virtual Subgraph marker、draft/editorStore、cross-store cache、keep-alive singleton、mergePool 等 trap。

这些条目仍被 archived Topics 引用，因此没有删除；其 `read_when` 已限制到旧基线/归档取证，并加入不得回接 3.1 的说明。

### 仍需按触发条件复验

- `pkg/input` 下 PostMessage/SendInput/UE Slate 的旧实机 trap 可能仍影响当前 provider，但必须先对照当前实现，不能直接照搬旧修法。
- Vue Flow 相机、shallow sync、deep watch 等一般交互 trap 仍可能有效；旧 Container 文件名只作为事故 provenance。
- research/wayfinder 保留原文，不把开放问题或当时建议批量改写成当前事实。

## 框架健康结论

“当前框架没啥大问题，只是不够完善”只对 **执行内核** 成立，对整个产品架构不成立。

### 没有根本方向问题、应保留

- versioned Data Type / Node Contract / Catalog；
- Workflow Source → Compiler → Program → Executor 唯一路径；
- strict types、explicit conversions、typed State；
- capability/admission/Run Grant、resource lease；
- durable Run journal 与同 scheduler debugger；
- Source-native GraphCall/authoring command。

### 不只是功能缺失、必须结构性修复

- adapter descriptor → Host Profile capability 的第二事实源；
- target/settings/authoring/admission/provider 的 generation lifecycle 分裂；
- platform Adapter 没拥有 profile/schema/editor 全链；
- recording 没有形成单一 Session/invariant owner；
- Asset library 和 node picker 两条查询/身份链；
- frontend RPC transport 吞错并制造二次错误；
- 巨型 UI/application modules 使用户流程缺少 locality；
- Stage acceptance 以组件/数量/页面可见代替黄金旅程。

所以准确说法是：**内核架构健康且可继续投资；外围架构存在可定位、可重构的边界问题；产品能力还不完整。无需推翻重写，但也不能只补齐几个节点和 UI 就发布。**

