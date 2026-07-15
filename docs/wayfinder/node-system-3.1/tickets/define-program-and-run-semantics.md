---
title: 定义 Program 3.1 与 Run 语义
label: wayfinder:grilling
parent: ../map.md
status: closed
assignee:
blocked_by:
  - define-node-contract-metaschema.md
  - define-capability-and-target-planning.md
  - define-blob-store-streams-and-resource-broker.md
---

# 定义 Program 3.1 与 Run 语义

## Question

Compiler 应把 Data、Exec、Error、event、region、subgraph、disabled、retry、cancel、timeout、recorded value 与 lineage lower 成什么不可变 Program 事实，runtime 又应提供多小的 interpreter interface，才能彻底删除按 node kind 分支的 ContainerRunner 并保证调试与正常执行同义？

## Resolution

已接受。Yotta 采用“不可变 Program + admission + generational RunRecord + NodeAttempt journal + Run Owner”作为唯一运行模型，不把 workflow/container mutable state 带入执行：

1. **Program Snapshot** 只保存 Compiler 可确定的事实：完整 graph plan、typed binding、implementation locks、Catalog hash 与 attributed Capability Plan。Compiler 不读取本机 target、credential、approval 或 provider session。
2. **Admission** 严格按 `strict-open Program → plan target binding → policy seal Run Grant → durable create QUEUED RunRecord → notify Worker` 执行。任何 target/credential/consent/ABI 歧义都在 provider side effect 前失败；不存在 GUI、builtin 或本地脚本免授权分支。
3. **RunRecord** 是单 Run 的 durable 状态机，只允许 `QUEUED → RUNNING → SUCCEEDED|FAILED|CANCELLED|INTERRUPTED`，或 `QUEUED → CANCELLED`。每次变化产生新 generation 和 record digest，Run Store 以 previous digest 做 CAS 原子替换；进程重启把遗留 RUNNING 写成 INTERRUPTED，绝不自动重放 effect。
4. **Run Value** 在 Value Envelope 外保存 value ID、graph/node/port/attempt 与 envelope digest。Value Envelope 的摘要不包含 provenance；durable RunRecord 只接受 inline/blob artifact，stream/resource authority 无 artifact 因而不能持久化。
5. **NodeAttempt / AdapterAction** 是 Run 的追加型执行事实，记录 graph path、node/effect、attempt、稳定 error code、时间、Program/Catalog/Grant identity 与经统一 redactor 产生的摘要；raw error、secret、prompt、路径与 bearer material 不进入 durable projection。
6. **Run Grant** 精确绑定 Program/plan/Run/principal/policy generation 以及每个 graph/node/requirement 的 provider、target、resource kind、plugin instance、session、operation、canonical scope 和 credential binding metadata。Grant projection 不含 bearer secret；每次 Broker open/borrow/call 都重新授权，通用 borrow 只接受 exact canonical scope。
7. **Run Owner** 是一个 admitted Run 全部临时 authority 的 composition owner。终止顺序固定为 revoke Grant、cancel Run context、revoke Run leases、永久 close Broker；provider object 只存在于 Broker 内。
8. **Interpreter** 最终只消费 strict-open Program、Run-scoped invocation context 与 narrowed capability session。retry/timeout/cancel/error routing 必须由 Program instruction 和 attempt state machine 表达；adapter 不能访问 ambient ServiceBundle 或自行重编译 Source。

`internal/runid`、`internal/capability.RunGrant`、`internal/admission` 与 `internal/run.Record/Store/Owner/GrantAuthorizer` 已实现上述 identity、target/policy admission、状态、CAS、durable Grant recovery、重启中断、durable value、NodeAttempt/AdapterAction/NodeStatus journal 与 resource lifecycle 内核。Compiler 已把 Data binding 与有序 Exec/Error route 分别 lower 成 `dataOrder` 和 `signalRoutes`，严格 opener 从可信 Catalog 重验两类指令；控制环不参与 data 拓扑。Executor 的确定性 scheduler 已实现 pull-data dependency、显式 exec selection、结构化 error routing、真实 attempt provenance、invocation budget、status declaration 校验与 per-invocation resource session；adapter 必须主动记录每个 declared effect，缺失、重复、未声明 action/error/status 均 fail closed。尚未完成的运行链工作是 production interpreter composition 与旧 ContainerRunner 删除；这些缺口不得由 dual-write、legacy read 或 fallback runtime 填补。
