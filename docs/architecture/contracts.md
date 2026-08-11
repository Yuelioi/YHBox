# Contracts and generated projections

Yotta 的 Source、Data Type、Node、Program、Run、package 与 transport 都有独立 identity。产品版本不会替这些
合同决定兼容性；当前值可由代码统一列出：

```powershell
task versions:inventory
```

## Contract chain

```text
Data Type definitions
        │
        ▼
versioned Node Contracts + implementation locks
        │
        ├──> sealed Node Catalog ──> Compiler / runtime adapters
        │
        └──> Authoring Projection ──> GUI / AI / MCP / documentation

Workflow Source schema ──> authoring patch schema ──> Program ──> Run Record
```

主要所有者：

| Contract | Go authority | Tracked projection |
| --- | --- | --- |
| Workflow Source / authoring patch | `internal/workflow/schema`、`internal/workflow/authoring` | `contracts/workflow/` |
| Data Type | `internal/datatype` | Workflow/Node schema 中引用的 type contract |
| Node Contract | `internal/nodecontract`、`internal/nodes` | `contracts/node/` |
| Node Catalog / Authoring Projection | `internal/nodecatalog`、`internal/nodeauthoring` | `contracts/node/current/` |
| Program | `internal/workflow/compiler` | runtime artifact；identity 由 package 定义 |
| Run Record | `internal/run` | Run Ledger 中的 canonical artifact |
| Plugin protocol / SDK | `internal/pluginprotocol`、`sdk/plugin` | `contracts/plugin/` |
| Wails RPC | registered Go services | `contracts/wails-rpc.json` |

`frontend/bindings/` 是 Wails 的本地生成物并被 gitignore；不能手改或提交。Tracked contract 是审查输入，
但生成它的 Go 类型、Catalog assembly 与 generator 才是事实来源。

## Node identity

Node identity 不是标题或图标。一个可执行 node 由稳定 `nodeTypeId`、独立 SemVer、semantic digest 和精确
implementation lock 共同确定。Contract 同时描述：

- data/exec/error port 与 exact type expression；
- config JSON Schema、instruction、retry 与 state access；
- capability requirement 或 Configured Target operation；
- declared error/status event；
- implementation ABI、entrypoint、version 与 conformance identity。

Compiler 必须在执行前把 Source 中的 exact NodeRef 解析到已 seal 的 Catalog/implementation。不能按标题、
category 或“最新版本”猜测实现。

## Generated views

```powershell
task contracts:check      # 内存重生 Workflow/Node artifacts 并拒绝 drift
task contracts:update     # 明确接受并写回新的 tracked artifacts
task plugins:check        # 重生 plugin Proto/WIT/SDK/conformance views
task plugins:update       # 明确接受 plugin contract 更新
task check:bindings       # 重生 Wails bindings 并核对 tracked RPC contract
```

生成文件变化必须和所属 Go contract 变化一起审查。不要只编辑 JSON/Markdown projection 来“修复”drift；也不
要在 README 复制节点、method、model 或 port 数量，因为它们是一次生成快照。

## Published compatibility floors

`contracts/workflow/v*` 与 `contracts/node/v*` 是 schema/projection 版本；它们不是“某个产品版本实际发布了哪些
引用”的证明。公开发布事实由以下 create-only snapshot 记录：

| Snapshot | Protects | Gate |
| --- | --- | --- |
| `contracts/releases/<product>/version-domains.json` | 已发布 durable/portable/contract/protocol writer 版本仍在 reader 支持集合中 | `task versions:compatibility:check` |
| `contracts/node/releases/<product>/builtin-node-refs.json` | 已发布 NodeRef 仍精确存在或有完整相邻 migration 链 | `task nodes:compatibility:check` |
| `contracts/catalog/releases/<product>/builtin-catalog-refs.json` | 已发布 TypeRef/CapabilityRef 的 ID 与 semantic digest 未被移除或原地改变 | `task nodes:compatibility:check` |

新增发布 floor 使用 `task versions:compatibility:freeze` 与 `task nodes:compatibility:freeze`。`task package` 会以
release 模式要求当前 `VERSION` 的 snapshot 已存在；snapshot 一旦形成就不得覆盖。版本域声明只是门禁输入，
reader 是否真实可用仍必须由冻结旧 fixture、持久迁移和关闭重开测试证明。

Breaking-change 与 migration 规则见[Compatibility and migration policy](../compatibility.md)，新增节点的实际
步骤见[Node development knowledge](../../flightdeck/knowledge/nodes/development.md)。
