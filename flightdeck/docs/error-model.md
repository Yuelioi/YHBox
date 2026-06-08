---
status: active
when_to_read: 给节点加错误处理 / 加新错误码 / 改 dispatch 失败路由 / 加 region 容错 / 撞「节点报错没被 Fail 出口接住」类问题前
applies_to: [error-model, node-framework, dispatch, Failf, NodeError, Coded, errorcode, Fail-output, Throw, region, validator]
when_to_update: 改 dispatch 失败路由 / Failf / NodeError / Coded 语义 / 集中错误码表 / Fail 出口约定 / region 容错策略时
last_updated: 2026-06-07
---

# 错误模型：节点失败出口 + Coded 路由 + 集中错误码

> 取代了旧的「异常冒泡 + Try 包子图」模型（Try 节点已删）。设计/落地过程见 archive 的
> `2026-06-07-error-model-per-node-fail-output` spec+plan。

## 一句话

节点 `Run` 返错时，框架**只把实现 `node.Coded` 接口的错误**（= 显式 `node.Failf` / `Throw`）路由到该节点**接了线的 `Fail` exec 出口**走失败分支；没接线、或返裸 `fmt.Errorf`（配置错）、或控制哨兵（Break/Continue/Stop/取消）→ 照旧冒泡中断。"整块兜底" 由 region 节点（Subgraph/Loop/CollapsedNode）的 Fail 出口承担。

## 三个核心概念（`internal/node/errorcodes.go`）

1. **`ErrCode`**（snake_case 字面值）+ **集中注册表 `ErrorCodes`**：`launch_failed` / `capture_failed` / `write_failed` / `not_found` / `timeout` / `playback_failed` / `send_failed` / `thrown` + 兜底 `error`。常量 CamelCase（`CodeCaptureFailed` 等）。注册表是「推荐全集」，非强约束——用户 Throw 自填码、未来插件返非注册码仍合法。
2. **`Coded` 接口**：`interface{ ErrCode() ErrCode }`。**dispatch 失败路由的准入闸**——只有实现它的错误会被路由。`*NodeError` 和 `*ThrowError` 实现它。
3. **`NodeError` + `Failf`**：
   ```go
   type NodeError struct { Code ErrCode; Message string; cause error }
   func (e *NodeError) Unwrap() error    { return e.cause }   // 保错误链
   func (e *NodeError) ErrCode() ErrCode { return e.Code }
   func Failf(code ErrCode, cause error, format string, args ...any) error
   ```
   **`cause` 显式传**（不靠调用方记得 `%w`），所以 `errors.Is(err, 底层哨兵)` 仍可追根因。迁移写法：
   `fmt.Errorf("DetectColor: %w", err)` → `node.Failf(node.CodeCaptureFailed, err, "DetectColor: %v", err)`。

## 配置错 vs 运行时错的边界（关键设计）

dispatch **不猜**错误类型，靠类型机制分：

| | 用什么返 | 实现 Coded? | 失败路由? |
|---|---|---|---|
| **运行时错**（IO/截屏/输入发送/找窗/起程序/回放——只有真去操作才知） | `node.Failf(code, cause, …)` | ✅ | ✅ 可被 Fail 出口截 |
| **配置错**（忘填变量名/空 Target/非法 axis/unknown mode/parse 失败——validator 该拦） | 裸 `fmt.Errorf` | ❌ | ❌ 永远冒泡中断 |
| **控制哨兵**（Break/Continue/Stop/ctx 取消） | 各自 sentinel | ❌ | ❌ 在 Coded 检查前被拦 |

> 给节点加错误时，先判这是「运行时故障」还是「图配置错」。前者 `Failf` 带码，后者保持裸 `fmt.Errorf`（让 validator 在编辑期拦）。

## dispatch 失败路由（`internal/services/container/runtime/dispatch_v5.go` → `routeResult`）

`result.Error != nil` 时的判定顺序（**顺序是正确性关键**）：

1. `control.IsStopRequested` → 停 run。
2. `errors.Is(ctx.Canceled)` → 透传（取消不是失败）。
3. `control.IsBreakRequested / IsContinueRequested` → 冒泡（**必须漏给 `Loop.RunRegion`，绝不进失败路由**）。
4. **失败路由**：`errors.As(err, &coded)`（命中 Coded）**且** `r.edges.has(node.ID+".Fail")`（该节点 Fail 出口有出边，O(1) 查 `edgeIndex.out` map）→ 合成失败分支 `nextWithData(node.ID+".Fail", …, {"Error":msg, "Code":code})`，**不冒泡**。
5. 否则（裸 error / 没接线）→ 冒泡。

事件：始终 emit `container:node-error`，带 `handled bool`（路由成功=true，柔和；冒泡=false，红高亮）+ `code`。⚠️ **目前前端无该事件的消费者**（无节点错误高亮 UI）——字段已就绪，留待将来。

**region 兜底是同一套机制**：Subgraph/Loop/CollapsedNode 的 `RunRegion` 裸 `return nil, err` 透传 body 错误（不 wrap，Coded 链完整），经同一个 `routeResult`（region 节点走 `execNodeAsRegionViaFramework` 也调它）路由到 region 自己的 Fail 出口。错误沿 RunRegion 调用栈逐层向外，**首个声明且接线的 Fail 出口消费**；中间没接线的 region 透传继续外冒。

## 谁拿 Fail 出口（7 个）

`OutputSpec{Name:"Fail", Type:"Exec", Semantic:"error", Data:[{Error,String},{Code,String}]}`。

- **curated 4 个**（值得就地处理的独立失败）：RunProgram(`launch_failed`)、Screenshot(`write_failed`/`capture_failed`)、WindowTarget(`not_found`)、PlayClip(`playback_failed`)。
- **region 3 个**：Subgraph、Loop、CollapsedNode（透传内部 Code）。

**其余 ~15 个运行时可失败节点不拿引脚**（DetectColor/模板/输入类等）——失败多是「目标窗没了」同根因，靠整块 region 兜；但**仍改 `Failf` 带码**，冒到 region Fail 出口时 Code 有意义。

> ⚠️ **WaitWindow 不拿 Fail 出口**：它「找不到窗」走的是 **Timeout 结果分支**（不是 error），`return nil, err` 只剩配置错/取消。结果分支 ≠ 错误——别给它加 Fail（否则是永不触发的死引脚）。这是「预期内的另一种结果用结果分支、error 专指意外故障」原则的体现。

## 消费 Error/Code（数据线 → 下游 data-in）

Fail 出口的 `Error`/`Code` 既沿 **Fail exec 边**作为 exec-data 下发，又被前端暴露成**可拉 data 线的数据出口**（`splitOutputs` 把 exec 出口的 `Data` 字段收进 `dataOut`）。把 `节点.Code` 拉一条 data 线到下游某个 data-in（典型：`Switch.Value` 按错误码分流、`Log.Message` 显示错误串），运行时这样解析：

- **validator**：`dataOutPinTypeForKind` 把「exec 出口下的 `Data` 字段」也算 data-out（`IsDataOutPin` 因此为真）→ INVALID_PIN / 类型校验 / data-graph DAG / sentinel 全部正确级联。配套谓词 `IsExecOutputDataField(kind,pin)` 标出「这是 exec 出口的数据字段」。
- **runtime**：这类 pin **不走 pure-data pull**（`evalDataSource` 只认 `IsPureData` 源）。`pullDataPin` 撞到 `IsExecOutputDataField` 的源直接返 nil；真正的值由 `ContainerRunner.applyExecDataEdges`（`dispatch_v5.go`，普通 + region 两条 exec 路径都调）从 token 带下来的 **exec-data** 按字段名取，回填进 dataWire。即「值已经沿 Fail exec 边流到下游了，data 线只是把 `Code` 重映射到目标 pin 名（如 `Value`）」。

> ⚠️ **约束：data 线必须跟 Fail exec 边并行**——源节点 = 触发本节点的 exec 上游。因为 exec-data 只承载「触发本次执行的那条 exec 边」的字段；若把 `Code` 拉给一个不在该失败分支上的节点，exec-data 里没这个 key → 该 pin 取不到值（静默 nil）。validator 暂不强制这条并行性。

## Throw（`internal/nodes/system/sentinels.go` + `throw.go`）

`ThrowError{Message, Code}` 实现 `Coded`（空 Code → `thrown`）。Throw 节点有可选 `Code` 输入（自由文本，前端给注册表码做下拉建议）。因实现 Coded，Throw 抛的错**走失败路由**，被最近接线的 region Fail 出口截；一路没接 = 抛到顶中断。（Break/Continue 反之，是控制哨兵，永不被截。）

## 前端（数据驱动，几乎零硬编码）

- 节点 spec 经 `GetAllNodeSpecs` RPC 透传 `OutputSpec.Semantic`；adapter（`nodeRegistry/adapter.ts` `splitOutputs`）把 `semantic==='error'` 的出口名收进 `NodeKindSpec.errorOut`。
- `ContainerFlowNode.vue` `execOutPinsForRender`：`errorOut` 成员标 `isError` → 渲染**红 exec 引脚**（`#f87171` + 红辉光）。label 统一走 `t('common.fail_pin')`（「失败」/「Fail」）。**Subgraph 例外**：它的 exec 出口本来只从子图 outputPins 派生，需单独把 spec 的 Fail 出口补进渲染列表（否则 Subgraph 的 region 兜底在 UI 上连不上）。
- 错误码注册表经 `NodeService.GetErrorCodes()` RPC，boot 时进 `stores/nodeRegistry.ts` 的 `errorCodes`。SwitchInspector 的 case 值输入用它做下拉建议（自由文本仍可填）→ 下游可按错误类型分流。
- i18n：`errorcode.<code>`（zh/en）做码的中文标签。

## 加东西怎么做

- **加一个错误码**：`errorcodes.go` 加常量 + 进 `ErrorCodes` 注册表 + `frontend/src/i18n/{zh,en}.ts` 的 `errorcode.<code>` 加标签。守卫测试 `TestAllErrCodesRegistered` 会逼你注册。
- **给已有节点加 Fail 出口**：Spec.Outputs 加那段 `Semantic:"error"` 模板 + 把运行时 error 改 `Failf` 带码。dispatch/前端零改动（数据驱动）。
- **新节点要不要 Fail 出口**：先判失败是「预期内结果」(→结果分支) 还是「意外故障」(→Failf)；故障值不值得就地处理(→Fail 引脚) 还是靠 region 兜(→只 Failf 带码不加引脚)。

## 关键文件

| 文件 | 职责 |
|---|---|
| `internal/node/errorcodes.go` | ErrCode/注册表/Coded/NodeError/Failf |
| `internal/node/spec.go` | `OutputSpec.Semantic` 字段 |
| `internal/node/service.go` | `GetErrorCodes` RPC |
| `internal/nodes/system/sentinels.go`/`throw.go` | ThrowError(Coded) + Throw 节点 |
| `internal/services/container/runtime/dispatch_v5.go` | `routeResult` 失败路由 |
| `internal/services/container/runtime/runner.go` | `edgeIndex.has` |
| `frontend/.../nodeRegistry/adapter.ts` `index.ts` | `errorOut` 派生 |
| `frontend/.../ContainerFlowNode.vue` | 红引脚渲染 |
| `frontend/.../inspector/SwitchInspector.vue` | 错误码下拉建议 |

## 已知缺口 / backlog

- 前端无 `container:node-error` 消费者 → 没有「节点失败/已处理」的画布高亮（事件字段已就绪，要建订阅+高亮子系统）。
- 示例容器（fishing-v2 等）仍引用已删的 Try kind，打开/运行会报未知 kind，未迁移。
- Throw 语义 breaking change：旧图里 Throw 外层若恰有接线 region Fail 出口，升级后 Throw 会被截获而非终止（静默变化）。
