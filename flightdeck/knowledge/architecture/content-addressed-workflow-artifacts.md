---
kind: note
summary: "Yotta 3.1 与 Workflow/Node/Data/Program 3.1 统一发布命名，但内容摘要算法域独立版本化；canonical DTO 改变必须换 hash domain 并拒绝旧 artifact，不能保留双 runtime。"
activation: action
read_when: "修改 Workflow Source 数字/JSON 边界、CatalogSnapshot、Compiler、ProgramSnapshot、ProgramStore、执行队列或 runtime bind 时。"
recheck_when: "RFC 8785 实现、hash 算法/域、Program format、NodeContract projection、compiler build identity 或插件 implementation lock 改变时。"
---
# Content-addressed Workflow artifacts

## 产品版本与协议代际

Yotta 3.1 是产品发布版本，Workflow/Node/Data/Program 3.1 是同一发布线内部的协议代际。当前代码中的 v3 Source/Catalog/Program 仍是未发布的 Compiler 切片：其 strict parse、JCS、diagnostic budget、opaque seal 与 trusted reopen 可以复用；一旦 3.1 改变 canonical DTO、类型身份、contract projection 或 execution plan，就必须使用新的 format/hash domain，并在 strict boundary 拒绝 v3。禁止在同一 domain 下改变摘要含义，也禁止让 v3/3.1 两个 runtime 同时成为生产事实。

Data Type 的 semanticDigest preimage 必须排除 digest 字段本身和 presentation annotations；Value digest 包含完整 Resolved Type。发布合同统一叫 3.1 不代表摘要算法域也叫 `/v3.1`：Product SemVer 不参与内容身份，Data Type 3.1 首版语义摘要仍使用冻结的算法域 `yotta/data-type-semantic/v1`。只有 canonical preimage 或摘要算法改变时才升级该域。

## 当前已实现的 v3 基础

三个身份不能混用：

- `sourceHash = SHA-256("yotta/source/v3\0" || JCS(source))`
- `catalogHash = SHA-256("yotta/catalog/v3\0" || JCS(machine catalog))`
- `programHash = SHA-256("yotta/program/v3\0" || JCS(program body))`

digest 只接受 `sha256:` 加 64 位小写 hex。JSON 必须是 UTF-8、无 duplicate key、可由 RFC 8785 表达；跨语言整数限制在 `±(2^53-1)`，禁止浮点 overflow/underflow。用户 Source 数组保序；capability 与 node lock 等 set 先排序去重。任何 domain 或 canonical DTO 变化都是 breaking contract，必须更新 golden hash。

`internal/workflow/catalog.Snapshot` 只投影编译相关 machine contract，排除 category、presentation widget、enum label/meta、i18n 与派生展示 badge；Point/Geometry 原先借 widget 表达的差异会提升成显式 machine `shape`，不能丢失。当前 in-tree implementation 由必填的整体 content digest `implementationSet` 锁定；`ContractHash` 只表示节点 machine contract，禁止冒充 executable implementation digest。

`ProgramSnapshot` 零值 invalid，没有公开 constructor；Compiler 私有 seal 与 `OpenProgram` 使用相同的 byte/depth 上限，保证成功编译的 artifact 可被同一 build 重新打开。所有 compiler diagnostics（含 parse/binding/capability）共享确定性的全局数量上限；schema 在正式验证前先从 Go model/jsonschema tags 反射派生 resource preflight（不维护第二份字段表），只识别“确定无效结构”，避免 validator 为 diagnostic bomb 构造完整 error tree，schema semantic phase 达限后立即停止。最终结果保留 canonical-sorted 前缀并把 `DIAGNOSTIC_BUDGET_EXCEEDED` 固定在最后；config map 必须先按 key 排序，不能因 Go map iteration 产生随机子集。`OpenProgram` 必须同时拿到可信 CatalogSnapshot 与 expected compiler build digest，并限制 collection budget、重验 canonical bytes、unknown/missing 字段、集合顺序、graph/node/lock invariant、binding/capability 与 program hash。所有 byte/slice accessor 返回副本。Hash 只证明 artifact 未改变，不能代替签名、ProgramStore ACL、permission grant 或 provenance。

Source 的 `requestedCapabilities` 是作者声明的能力上限；Compiler 从实际节点推导 `requiredCapabilities`，二者必须精确相等，缺少或多余都产生 stable Diagnostic。v3 子图只允许 compiler-owned `core.call-subgraph`，`core.*` 命名空间禁止 Catalog 节点注册：callee 用静态 `config.graphId` 指向同一 Source 内的 subgraph，GraphPort 的 `ID` 是稳定 machine identity、`Name` 仅展示，boundary endpoint 固定为无歧义的 `<nodeId>.<portId>`（两段均禁止 `.`）。Compiler 验证每个 boundary 已实际接线、方向/类型正确、恰好一个 exec entry 且至少一个 exec output；data output 也作为 typed caller output 冻结，并且只能有一个 producer。missing/main target 与任何静态递归均拒绝，只 lower 从 main 可达的 closure；不可达图不得进入 Program、NodeLocks 或 required capabilities。Program v3 的 call plan 自足保存每个 entry/input/output 的 boundary NodeID、PortID 与 Type，展开总量同时受 port 数与 estimated encoded bytes 预算约束，seal 后 artifact 也不得超过 Open 上限。`OpenProgram` 必须用可信 Catalog/build 重新 bind/lower 并逐图比较；Runtime 只按 GraphID 直接取已 lower graph 与冻结 endpoint，不得扫描 interface 或解释 legacy `Subgraph/CollapsedNode/Params/Normalize`。

`NodeSpec.DynamicPorts` 是进入 NodeContract/Catalog hash 的 closed descriptor，不再用三个 bool 暗示运行时另有 parser。当前 Compiler 只接受 `role=output + shape=names + fixedType=Exec`，由 Switch 的 `config.cases` 生成有序端口：1–256 项、单名最多 128 UTF-8 bytes、精确判重，拒绝空白边界、`.`、控制/双向控制字符和静态出口冲突；全编译最多 10,000 个 resolved dynamic ports。Program v3 为每个节点保存必填 `dynamicPorts`（静态节点为 `[]`），`OpenProgram` 先做同预算结构校验，再从可信 Catalog + config 重派生并 exact compare。按节点解析后的 pin index 禁止进入按 kind cache，不同 Switch 的 cases 必须隔离。dynamic input、output data、effect 和外部依赖在对应 declarative phase 完成前仍明确拒绝，不得降级或绕过。

`InputSpec.Constraints` 是进入 NodeContract/Catalog v3 identity 的声明式输入约束，当前只接受 `nonBlank` 与 `numberGreaterThan`。数值 threshold 虽以 string 保存以避免 Go/JSON 精度漂移，进入 Registry 或 Catalog 投影前仍必须按 RFC 8785 数字语义 canonicalize；只排序 constraint 而保留 `0`/`0.0` 等不同拼写会让等价 contract 获得不同 hash。默认值在 Catalog 构造时校验，Source 中未接线的静态值在 Compiler 中校验，接线值延迟到 runtime adapter；全图 constraint evaluation 有独立预算。声明式 constraints 与 legacy `Validator` callback 不得并存，避免同一规则出现两个事实源。

新 Compiler 不得 import legacy container runtime/store/execution queue。旧 Runtime 迁移前仍是独立路径；不得把 `Container.Normalize()`、按 ID 读 Store 或持 mutable graph pointer 的 `CompiledContainer` 引入新 Compiler。

