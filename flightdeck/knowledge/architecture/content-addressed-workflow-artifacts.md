---
kind: note
summary: "Yotta/Workflow/Node/Data/Catalog/Program 统一为 3.1；内容摘要算法域独立版本化，strict boundary 必须用可信依赖重新验证。"
activation: action
read_when: "修改 Workflow Source、Data Type、Node Contract、Catalog、Compiler、Program、ValueEnvelope、执行绑定或 implementation lock 时。"
recheck_when: "RFC 8785 实现、hash preimage/domain、Program format、NodeContract projection、compiler build identity 或插件 implementation lock 改变时。"
---
# Content-addressed Workflow artifacts

Yotta 3.1 是产品版本，也是当前 Workflow/Node/Data/Catalog/Program contract generation。摘要算法域不跟产品 SemVer 机械同步；只有 canonical preimage 或算法改变才升级 `/vN` domain。改变 durable DTO 时必须换 format/domain 并拒绝旧 artifact，不能在同一 domain 下改变摘要含义，也不能长期运行双 runtime。

当前身份：

- Data Type semantic digest：`yotta/data-type-semantic/v1`，preimage 排除 digest 自身和 authoring metadata。
- Node Contract semantic digest：`yotta/node-contract-semantic/v1`，preimage 排除 authoring metadata。
- Workflow source hash：`yotta/workflow-source/v1` + RFC 8785 source。
- Catalog hash：`yotta/catalog/v1` + canonical machine catalog。
- Program hash：`yotta/program/v1` + canonical Program body。
- Value digest：`yotta/value-envelope/v2`，preimage 包含 Value Envelope version、完整 Resolved Type、representation、codec 和 canonical value。

所有 digest 只接受 `sha256:` 加 64 位小写 hex。JSON 必须是 UTF-8、无 duplicate key、可由 RFC 8785 表达；跨语言整数限制在 `±(2^53-1)`。任何会进入 recursive canonicalizer、validator 或 typed decoder 的不可信 JSON，必须先经过迭代式 byte/depth/node/string budget。

`internal/nodecatalog.Snapshot` 保存 sealed Data Type、Node Contract 与每节点 implementation lock；presentation 是独立 artifact/digest。implementation lock 在 host dispatch 时必须完整匹配 package、artifact digest、ABI 与 entrypoint，Node semantic digest 不能冒充 executable implementation digest。explicit manifest version 必须随内建实现或 ABI 行为改变而升级。

`ProgramSnapshot` 零值 invalid、无 public constructor。`OpenProgram` 同时需要可信 Catalog 与 expected compiler build，并重验 canonical bytes、hash、source/catalog/build identity、entry graph、collection budget、exact node/implementation lock、effective ports/execution、config、typed input envelope、edge endpoint/type、topological order与 capability manifest。所有公开 byte/slice view 返回副本。Hash 只证明 artifact 未改变，不能代替签名、ACL、capability grant 或 provenance。

Source 的 `requestedCapabilities` 是作者声明上限，Compiler 从实际 contract 推导 `requiredCapabilities`，两者必须精确相等。Interpreter 仍必须取得宿主 grant；Program 声明不能自我授权。当前 3.1 tracer 只实现 pure-data main graph/data edge/inline literal/default，其他 Source feature 必须 fail closed，不能接受后静默丢弃。

ValueEnvelope 是 Program/host 的值边界。Program literal 保存 `literal`/`default` provenance 和 envelope artifact；host adapter 只取得验证后的 payload，返回值必须按 pinned Data Type schema 验证后再封装。单值、Program 与整次运行保留值分别有资源预算。

Blob 内容身份使用原始字节的 SHA-256，不使用 JSON artifact 的 domain-separated `artifact.Sum`；Blob Reference 另含 canonical media type 与 exact size。Blob/inline envelope 可持久化，Stream/Resource envelope 是 Run-only authority，禁止进入 Program、durable trace、日志、clipboard 或 cache。Resource token 必须由 Broker 以 256-bit randomness 签发并绑定 Run/invocation/operation/expiry；内容 hash 不能充当 authority。

新 Compiler 不得 import legacy container runtime/store/execution queue。旧 `ContainerRunner` 迁移前仍是独立生产路径；3.1 interpreter 不得作为 fallback。Program/Run、Capability/Resource 与 catalog-wide migration 完成后整体删除旧 runtime。
