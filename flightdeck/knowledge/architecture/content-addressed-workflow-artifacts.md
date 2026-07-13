# Content-addressed Workflow artifacts
SUMMARY: Workflow v3 的 Source、Catalog 与 Program 使用 RFC 8785 canonical JSON、版本化 hash domain、strict digest 与 opaque seal；hash 是完整性标识，不是签名或授权。
READ WHEN: 修改 Workflow Source 数字/JSON 边界、CatalogSnapshot、Compiler、ProgramSnapshot、ProgramStore、执行队列或 runtime bind 时。
RECHECK WHEN: RFC 8785 实现、hash 算法/域、Program format、NodeContract projection、compiler build identity 或插件 implementation lock 改变时。

---

三个身份不能混用：

- `sourceHash = SHA-256("yotta/source/v3\0" || JCS(source))`
- `catalogHash = SHA-256("yotta/catalog/v1\0" || JCS(machine catalog))`
- `programHash = SHA-256("yotta/program/v2\0" || JCS(program body))`

digest 只接受 `sha256:` 加 64 位小写 hex。JSON 必须是 UTF-8、无 duplicate key、可由 RFC 8785 表达；跨语言整数限制在 `±(2^53-1)`，禁止浮点 overflow/underflow。用户 Source 数组保序；capability 与 node lock 等 set 先排序去重。任何 domain 或 canonical DTO 变化都是 breaking contract，必须更新 golden hash。

`internal/workflow/catalog.Snapshot` 只投影编译相关 machine contract，排除 category、presentation widget、enum label/meta、i18n 与派生展示 badge；Point/Geometry 原先借 widget 表达的差异会提升成显式 machine `shape`，不能丢失。当前 in-tree implementation 由必填的整体 content digest `implementationSet` 锁定；`ContractHash` 只表示节点 machine contract，禁止冒充 executable implementation digest。

`ProgramSnapshot` 零值 invalid，没有公开 constructor；Compiler 私有 seal 与 `OpenProgram` 使用相同的 byte/depth 上限，保证成功编译的 artifact 可被同一 build 重新打开。`OpenProgram` 必须同时拿到可信 CatalogSnapshot 与 expected compiler build digest，并限制 collection budget、重验 canonical bytes、unknown/missing 字段、集合顺序、graph/node/lock invariant、binding/capability 与 program hash。所有 byte/slice accessor 返回副本。Hash 只证明 artifact 未改变，不能代替签名、ProgramStore ACL、permission grant 或 provenance。

Source 的 `requestedCapabilities` 是作者声明的能力上限；Compiler 从实际节点推导 `requiredCapabilities`，二者必须精确相等，缺少或多余都产生 stable Diagnostic。v3 子图只允许 compiler-owned `core.call-subgraph`，`core.*` 命名空间禁止 Catalog 节点注册：callee 用静态 `config.graphId` 指向同一 Source 内的 subgraph，GraphPort 的 `ID` 是稳定 machine identity、`Name` 仅展示，boundary endpoint 固定为无歧义的 `<nodeId>.<portId>`（两段均禁止 `.`）。Compiler 验证每个 boundary 已实际接线、方向/类型正确、恰好一个 exec entry 且至少一个 exec output；data output 也作为 typed caller output 冻结，并且只能有一个 producer。missing/main target 与任何静态递归均拒绝，只 lower 从 main 可达的 closure；不可达图不得进入 Program、NodeLocks 或 required capabilities。Program v2 的 call plan 自足保存每个 entry/input/output 的 boundary NodeID、PortID 与 Type，展开总量同时受 port 数与 estimated encoded bytes 预算约束，seal 后 artifact 也不得超过 Open 上限。`OpenProgram` 必须用可信 Catalog/build 重新 bind/lower 并逐图比较；Runtime 只按 GraphID 直接取已 lower graph 与冻结 endpoint，不得扫描 interface 或解释 legacy `Subgraph/CollapsedNode/Params/Normalize`。dynamic pin/data field、自定义校验、effect 和外部依赖在对应 declarative phase 建立前仍明确拒绝，不得降级或绕过。

新 Compiler 不得 import legacy container runtime/store/execution queue。旧 Runtime 迁移前仍是独立路径；不得把 `Container.Normalize()`、按 ID 读 Store 或持 mutable graph pointer 的 `CompiledContainer` 引入新 Compiler。
