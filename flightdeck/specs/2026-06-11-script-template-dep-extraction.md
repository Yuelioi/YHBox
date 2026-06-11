---
status: active
summary: Script 节点扫 Code 里的模板/clip/subgraph GUID 字面量当依赖,堵住"脚本引用的资产被 GC 误删、库里删不警告"的盲区
last_updated: 2026-06-11
related: [specs/2026-06-11-script-call-subgraph.md]
---

# Script 资产依赖提取 (script-template-dep-extraction)

## 背景 / 为什么 (Stage 1, 单脚本路线的安全底座)

把钓鱼这类复杂逻辑搬进单个 Script 节点是**今天就可行**的(`while(true)` 主循环 + JS helper + 全部 ScriptBindable 节点可调,见 [docs/script-system.md](../docs/script-system.md))。但有一个**正确性盲区**必须先堵:

- 脚本里调 `CheckTemplate({Templates:["<guid>"]})` 时,GUID 是 `Code` 字符串里的字面量。
- 依赖扫描器 (`internal/services/container/dependency/scanner.go`) 靠每个节点的 `nodepkg.Get(kind).Dependencies(in)` 收依赖。**Script 节点没实现 `Dependencies()`** → 脚本里的资产引用对扫描器**完全隐形**。
- 后果两条:
  1. 库里「安全删除」资产时,`Dependencer` BFS 扫不到脚本 → **漏报 referrer**,用户以为没人用就删了。
  2. `gcBlobs` 的 live 集合(records 的 `variants[].blob` ∪ clip blob)不含脚本引用 → **把脚本在用的 blob 当孤儿回收**,运行时变 missing。

`internal/services/container/dependency/types.go:11` 已把 `KindScript` 列为预留扩展点——这就是兑现它。

## 方案

给 Script 实现 `Dependencies(in node.Inputs) []node.Dependency`:

- 从 `in.String("Code")` 取脚本源码,**正则提取 GUID 字面量**(`[0-9a-f]{8}-[0-9a-f]{4}-...` 标准 uuid 形),逐个返 `{Kind:"template", Key:guid}`(复用 CheckTemplate 的 `templateDeps` 同款 Dependency 形态)。
- clip GUID(`clip-<uuid>`)同理返 `{Kind:"clip", Key}`;subgraph 引用归 Stage 2 一起处理(见 related)。

**保守 over-approximation 是正确侧**:正则把脚本里**所有** uuid 字面量都标成依赖(哪怕某个其实是别的用途)。多标依赖 → 最多少删一个 blob(无害);漏标 → 误删(有害)。GC 安全永远选 over。

## 边界 / 取舍

- **不解析 JS 语义**:动态拼出来的 GUID(`"36" + rest`)扫不到。接受 + 在 script 文档写一句"模板/clip GUID 用字面量写,别字符串拼接,否则依赖追踪失效"。
- 模板 picker 在脚本编辑器里插入 GUID 时天然是字面量,正常写法都覆盖。
- 不区分 template/clip 的精度:可先统一按 uuid 提取、Kind 用 `template`(clip 的 `clip-` 前缀可判);若过粗导致 referrer 噪音再细化。

## 验收

1. 写一个 Script 引用某 template GUID → `ScanContainerDependencies` 的结果含该 template dep。
2. 库里删该 template → referrer 列表里出现这个容器(不再静默删)。
3. `gcBlobs()` 不回收该 GUID variants 的 blob。
4. 单测:`dependency/scanner_test.go` 加一例 Script-node-with-guid-literal。

## 关联

- [Stage 2: 脚本调子图](2026-06-11-script-call-subgraph.md) —— 那边脚本里 `Subgraph({SubgraphID:"<guid>"})` 的依赖也要被本提取覆盖(subgraph Kind),所以本 Stage 先行、把提取框架搭好。
