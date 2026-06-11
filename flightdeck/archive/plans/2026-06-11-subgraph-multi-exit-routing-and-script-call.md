---
status: done
summary: 先修 v5 runtime 子图多出口路由(按到达 decl 路由父图边, Subgraph/CollapsedNode 转 DynamicOutputs), 再把 Subgraph 暴露成脚本绑定函数 — 实施 script-call-subgraph spec 的 A 路线。两阶段全部落地, 验收 5 条全过。
last_updated: 2026-06-11
implements: archive/specs/2026-06-11-script-call-subgraph.md
---

# Subgraph 多出口路由 + 脚本调子图 (Stage 2)

用户拍板 A 路线 (2026-06-11): 先补子图多出口路由, 再做脚本调子图。

## 侦察实锤 (全部 grep/读源码确认, 2026-06-11)

1. **三套出口 key 并存, 互相对不上**:
   - runtime fire: `Subgraph/CollapsedNode.RunRegion` 写死 `ctx.Out("Done")` (静态 spec 出口)。
   - UI 边: 折叠 (`useFolding.ts:131`) 写 `callNode.<declID>`; `ContainerFlowNode.vue:232` 给 kind==='Subgraph' 渲 decl pin (id=decl.ID)。**→ UI 折叠出来的调用边今天根本不会被触发** (fire "Done" ≠ edge key declID)。
   - 手写数据 (fishing-v2): 边用 `.Done` (恰好匹配静态出口, 所以真机能跑); `callTryHookF.failed` 这类多出口边永远不触发。
   - validator (`validateInvalidPins`) 静态出口 + decl ID **都放行**, 所以不一致一直没暴露。
2. `runRegionBody` (dispatch_v5.go:387) 到达 output marker 直接 `return nil`, 丢弃到达的是哪个 decl。
3. `model.go:44` 约定: "对 Subgraph 调用节点, pinName = SubgraphOutputDecl.ID" — **decl ID 是既定约定**, 应以它为单一 key。
4. CollapsedNode 前端走静态 pin 渲染 (ContainerFlowNode 只特判 'Subgraph') — 跟 Subgraph 不一致, 要统一。
5. 全部现存子图体都有边连到出口 marker (fishing-v2 13 个 + UI 容器 1 个), entry marker `.Done` 播种约定不动。
6. `FindParentDownstreamByDeclID` (runtime/subgraph.go) 无非测试调用方 — 路由统一走 routeResult 后可删。
7. 脚本绑定层 (`services/script/binding.go`) 持 node.Ctx → ServiceBundle; RegionRunner 需要 runner 级状态 (dispatch table swap / frame stack / compiled.Subgraphs), 须由 runner 注入新服务。

## 设计决策

- **单一出口 key = decl ID**。Subgraph/CollapsedNode spec: `DynamicOutputs: true`, 静态出口只留 `Fail` (error 语义, 失败路由照旧)。删静态 `Done` (二号铁律: 不留兼容)。
- **body 契约改为 `func(Ctx) (exit string, err error)`** (registry.RunRegion 类型 / RegionRunner 接口 / RunNodeAsRegion 同步改): body 回报"region 到达了哪个出口"。Loop/ForEach 忽略 exit。Subgraph/CollapsedNode fire `ctx.Out(exit)`。
- **runRegionBody 返回到达的 decl ID** (output marker 命中时从 `currentSG.OutputDeclsByID[tok.NodeID]` 取 `.ID`); 队列跑干没到 marker → ""。
- **跑干没到出口的语义** (makeBodyForSubgraph 闭包里裁决): 单出口子图 → fallback `OutputPins[0].ID` (保持"未接 marker 的分支也继续"现行为, 录制自动折叠图依赖它); 多出口子图 → Coded 错误 `SUBGRAPH_NO_EXIT` (Fail 接线→失败路由, 否则冒泡) — 多出口跑干是图 bug, 静默挑一个出口是错误隐藏。
- **Loop body 内命中子图出口 marker**: 维持现状 (结束本轮迭代, 不穿透 Loop)。已知边界, 不在本次范围。
- **数据迁移**: 一次性脚本把 bin/data 全部容器里"源是 Subgraph/CollapsedNode 调用节点"的 `.Done` 边改成 callee 的对应 decl ID (fishing-v2 全部是 `done`)。不写 load-time shim。测试 fixture (`try_hook_f_test.go` 的 `call.Done`) 同步改。
- **脚本绑定 (Phase 2)**: 新服务接口 `node.SubgraphCaller` (`CallSubgraph(ctx, sgID, params) (exitName, error)`) 进 ServiceBundle + Ctx accessor, ContainerRunner 实现 (复用 makeBodyForSubgraph 抽出的共享核心: resolve sg → seed params(含 defaults) → push frame → swap tables → runRegionBody → restore)。脚本返回 `{exit: <decl Name>}` (人读名, 不是 UUID)。递归深度限制 (frame 深度 ≤ 32, 超出 Coded 错误)。watchdog/ctx 取消天然透传。
- ScriptBindable 判定不动 (Subgraph() 是定制绑定, 不走节点自动绑定)。

## Phase 1 — 子图多出口路由 (graph 层)

1. **框架契约**: `internal/node/` registry.go `RunRegion func(Ctx, Inputs, func(Ctx) (string, error)) (Outputs, error)`; interfaces.go RegionRunner 同步; engine.go RunNodeAsRegion 透传。受影响节点: Loop / ForEach (`_, err := body(c)`), Subgraph / CollapsedNode (fire body 返回的 exit)。全部相关单测改签名。
2. **Spec**: Subgraph / CollapsedNode → `DynamicOutputs: true`, Outputs 只留 Fail。
3. **runtime**: `runRegionBody` 返 `(reachedDeclID, error)`; `makeBodyForSubgraph` 闭包按上面语义裁决 exit; `makeBodyForLoop` 透传 err 忽略 decl; `disabled.go` 直通只走 `OutputPins[0].ID` (删 .Done 分支); 删 `FindParentDownstreamByDeclID` (+测试)。
4. **错误码**: `SUBGRAPH_NO_EXIT` 进 errorcode 表 (按 error-model doc 的加码流程)。
5. **前端**: ContainerFlowNode `execOutPinsForRender` 特判扩到 CollapsedNode (decl pin + Fail); 确认连线/补全无其他静态 Done 依赖 (grep 'Done' on Subgraph/CollapsedNode 路径)。
6. **数据迁移脚本**: 扫 bin/data/containers/**: from 节点 kind ∈ {Subgraph, CollapsedNode} 且 pin=='Done' → 改为 callee OutputPins 里 Name=='done' 或唯一 decl 的 ID; 迁完跑 validator 全绿。
7. **测试**: try_hook_F 多出口端到端 — done/failed 各路由到**不同** stop 节点并断言只走对应分支 (修 try_hook_f_test 原来两口同接一个 stop 的盲区); 单出口跑干 fallback; 多出口跑干 → SUBGRAPH_NO_EXIT; UI 风格 declID(uuid) 边路由; CollapsedNode 同套。

## Phase 2 — 脚本调子图

1. **服务接口**: `internal/node/` 新 `SubgraphCaller` interface + ServiceBundle 字段 + Ctx accessor (StubServices 置 nil-safe)。
2. **runner 实现**: 从 makeBodyForSubgraph 抽共享核心 `runSubgraphCall(ctx, sg, params, loopStack) (*SubgraphOutputDecl, error)`; CallSubgraph 包它 (含深度防护)。确认 listener subRunner (`listener.go:205`) 的 runner 实例隔离, 不与主 dispatch 抢 table swap。
3. **绑定**: binding.go Install 注入 `Subgraph({SubgraphID, ...params})` (服务非 nil 才装); 错误走 throwErr 约定; 返 `{exit: declName}`。
4. **依赖提取**: Stage 1 的脚本资产提取器扩 `Subgraph({SubgraphID:"…"})` → `Dependency{Kind:"subgraph"}` (ScanContainerDependencies 扫到被调子图)。
5. **前端**: scriptCompletions 加 Subgraph 函数项 (snippet `Subgraph({SubgraphID: "${}"})`) + 参考面板条目 + i18n。
6. **测试**: 脚本调单出口/多出口子图端到端 (exit 名/入参/取消); 递归深度限制; 依赖扫描; validator 不误报。

## 验收 (= spec 实现验收 5 条)

graph 层多出口端到端 / 脚本调子图 exit+参数+取消 / 停容器子图内阻塞节点取消 / ScanContainerDependencies 扫到脚本 SubgraphID / 防递归有明确行为。

## 收尾知识沉淀 (landing 时)

- script-system.md: Subgraph() 绑定约定; node-system-architecture.md / node-system-reference.md: RunRegion body 新契约 + Subgraph DynamicOutputs; error-model.md: SUBGRAPH_NO_EXIT。
- 2026-06-04-subgraph-marker-pin-convention incident: 出口 key 统一后按需补 Case。
