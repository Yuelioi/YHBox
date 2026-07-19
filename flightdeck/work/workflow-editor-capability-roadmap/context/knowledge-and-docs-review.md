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

| 3.0 | 3.1 | R5 |
| --- | --- | --- |
| Spec/init dispatch | versioned Contract + sealed Catalog | verified |
| Container runtime/vars | Source → Compiler → Program → Executor + typed State | verified |
| yt/Expr | isolated Script + typed authoring | remove |
| HWND target | installation slot + exact/regex resolver | verified native/UAC |
| capability 分支/启动快照 | Manifest + generation/lease/reclaim | verified |
| 旧 picker | paged Query/Picker + exact BlobRef | verified 1000×2 |
| 分裂录制 | Session → clip → asset → playback | verified native |
| Android/Browser 专链 | adapter-owned profile/manifest/runtime | verified |
| virtual subgraph | Source-native graph/interface/GraphCall | verified |

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

### R5 已退役的 3.0 主动 Knowledge

15 条无依赖旧知识已删除；3 条 archived Topic 固定引用的历史证据因 Flightdeck 归档不可变约束保留原路径，并严格收窄 `read_when`。主动路由不再把旧实现当现行规则。

### 仍需按触发条件复验

- `pkg/input` 下 PostMessage/SendInput/UE Slate 的旧实机 trap 可能仍影响当前 provider，但必须先对照当前实现，不能直接照搬旧修法。
- Vue Flow 相机、shallow sync、deep watch 等一般交互 trap 仍可能有效；旧 Container 文件名只作为事故 provenance。
- research/wayfinder 保留原文，不把开放问题或当时建议批量改写成当前事实。

## 框架健康结论

2026-07-18 审计准确：当时内核健康、外围未闭合。R1–R4 已重建 Typed RPC、Installation Manifest/Target Runtime、Recording Session、Asset Query/Picker 与 adapter-owned profiles，并用 Windows/Android/Browser 纵向复验。

R5 结论：**内核与外围架构均可继续投资，无需推翻重写；自动发布矩阵、UAC production 启动与最终黄金路径均已闭合。** 新平台仍只新增 Adapter/runtime，不修改 Source、Compiler、scheduler 或中央 ProfileDraft。
