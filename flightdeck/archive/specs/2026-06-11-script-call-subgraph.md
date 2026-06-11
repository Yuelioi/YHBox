---
status: done
summary: 把 Subgraph 暴露成脚本绑定函数 Subgraph({SubgraphID, ...params}) — 用户拍板 A 路线:先修好子图多出口路由(出口 key 统一 decl ID),再上脚本调子图。已全量落地(见同日 plan)。
last_updated: 2026-06-11
note: 用户拍板 A 路线并已落地 — 多出口路由(RunRegion body 回报出口 + DynamicOutputs + fishing-v2 数据迁移) + SubgraphCaller 服务 + Subgraph() 绑定 + 依赖提取 + 前端补全。实现验收 5 条全过(测试在册); 真机验证 2026-06-12 全过(fishing-v2 状态机流转 + 多出口子图脚本调用 exit/入参)。
related: [specs/2026-06-11-script-template-dep-extraction.md]
---

# 脚本调子图 (script-call-subgraph)

## 背景 / 为什么 (Stage 2)

当前 `node.ScriptBindable` (`internal/node/registry.go:130`) **排除 RegionRunner**(Loop/ForEach/Subgraph/CollapsedNode),理由:脚本有原生 for/while 循环,调子图"未实现、另立项"。

- 对**自包含**单脚本:helper 全用 JS 函数写就行,不需要这个。
- 对**可组合**编排:想让脚本当胶水、复用现有(及别人分享的)子图库当函数调,就需要这个。

这是真功能,不是补丁——所以独立成 Stage 2,只在出现"复用子图库"的实际场景时启动。

## 方案 (方向)

给脚本注入一个 `Subgraph({SubgraphID:"<guid>", <param>:<值>, ...})` 绑定函数,同步跑完目标子图、返 `{exit, ...outputs}`(同 exec 节点返回约定)。

复用 runtime 现成的子图执行机制 `makeBodyForSubgraph` (`internal/services/container/runtime/dispatch_v5.go`):推 ExecFrame + seed LocalParams + 切 dispatch table 到 callee + 跑下游 + 恢复。

## 难点 (为什么不是 Stage 1 顺手)

- 当前绑定层 (`internal/services/script/binding.go`) 走 `node.RunNode` / `node.EvaluatePureData`,只持 `ServiceBundle`(Vision/Input/Vars/...)。**RegionRunner 需要 runner 级上下文**(dispatch table、frame stack、预编译子图 `r.compiled.Subgraphs`),这些不在 ServiceBundle 里。
- 要把 runner 的"跑一个子图"能力下放给脚本 ctx——可能给 `node.Ctx` 或绑定层加一个"call subgraph by id"的入口,由 ContainerRunner 实现注入。这是核心设计活,需要单独 plan。
- 取舍预设:只支持**静态 SubgraphID + 单次同步调用**(同 Subgraph 节点语义,不接 dynamic data-in pin);脚本里**不开** Loop/CollapsedNode 绑定(原生 JS 循环替代,折叠节点是编辑期概念)。

## 依赖关系

- 依赖 [Stage 1: 资产依赖提取](2026-06-11-script-template-dep-extraction.md):脚本里 `Subgraph({SubgraphID:"<guid>"})` 的 subgraph 依赖也得被静态扫到(否则子图删除/分享导出闭包会漏),所以 Stage 1 把提取框架先搭好、本项扩一个 subgraph Kind 即可。

## 开工发现 (2026-06-11) — 撞前置缺口:子图多出口路由没接线

读源码确认实现路径时发现一个**比本 spec 更底层的缺口**,直接卡住"脚本调子图返回正确出口":

- **现状:v5 runtime 里 Subgraph 调用只会走单一 "Done" 出口,多出口(done/failed 之类)没路由。** 证据链:
  1. `runtime/subgraph.go` 的 `FindParentDownstreamByDeclID`(子图出口 decl → 父图下游边的唯一 helper)**没有任何非测试调用方**(repo-wide grep)。
  2. `dispatch_v5.go:runRegionBody` 到达 output marker 时直接 `return nil`,**丢弃是哪个出口 decl**。
  3. `nodes/system/subgraph.go:RunRegion` 写死 `ctx.Out("Done").Fire()`;且 Subgraph spec 只有静态 Done/Fail 出口(非 DynamicOutputs),fire 一个 "failed" 名会 panic。
  4. `try_hook_f_test.go` 把 `call.Done` 和 `call.failed` **都接到同一个 stop**、断言只看按键/变量 —— 多出口路由从未被测试区分。
- **连带**:fishing-v2 的 `try_hook_F.failed → RECOVERING`、`state_*` 里若干非 Done 子图出口,在当前 runtime 其实都**落不到正确分支**(真机快测没触发到那条罕见路径,故没暴露)。属 fishing 既有隐患,与本 spec 正交但同根。

### 需用户拍板的分叉

- **A. 先补「子图多出口路由」再做 Stage 2**(推荐):让 Subgraph 调用按到达的 output decl 路由到 `call.<declName>` 动态出口。这同时修好 graph 层的多出口子图(含 fishing 的 try_hook_F.failed)。Stage 2 在此之上自然返回正确出口。工作量中等、动 dispatch + Subgraph 出口模型。
- **B. Stage 2 先上「单 Done 出口」语义**:脚本 `Subgraph({...})` 只返 `{exit:"Done", ...}`,与当前 Subgraph 节点行为一致。能调用/复用单出口子图(press_esc 这类),但调多出口子图拿不到分支。等 A 落地再升级。

## 实现验收 (A 路线落地后)

1. graph 层:多出口子图调用按到达 decl 路由到对应父图下游(补 `FindParentDownstreamByDeclID` 接线 + runRegionBody 记录 reached decl)。
2. 脚本调一个多出口子图,端到端跑通,返回的 exit 名正确 + 输出 data + 入参传递都对。
3. 容器停止时子图内阻塞节点随 ctx 取消(同主脚本 watchdog 语义)。
4. `ScanContainerDependencies` 能从脚本扫到被调子图(Stage 1 的提取扩 SubgraphID pin)。
5. 防递归 / 防环(子图里又有 Script 调回自己)有明确行为(报错或深度限制)。
