---
status: active
summary: 把 Subgraph 暴露成脚本绑定函数 Subgraph({SubgraphID, ...params}),让脚本当编排层复用子图库(下一阶段,gated 在 Stage 1 之后)
last_updated: 2026-06-11
note: 下一阶段:自包含单脚本(JS helper 替代子图)已够用,本项仅当要复用子图库时启动;先做 Stage 1
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

## 验收

1. 脚本调一个现有子图,端到端跑通,出口名 + 输出 data + 入参传递都正确。
2. 容器停止时子图内阻塞节点随 ctx 取消(同主脚本 watchdog 语义)。
3. `ScanContainerDependencies` 能从脚本扫到被调子图(配合 Stage 1)。
4. 防递归 / 防环(子图里又有 Script 调回自己)有明确行为(报错或深度限制)。
