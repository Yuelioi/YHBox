# Storage consistency

Settings 使用 immutable snapshot：更新流程是 clone、mutate/merge、validate、atomic swap、atomic save。同目录临时文件写入并 sync，rename 后同步父目录；失败保留旧文件。

Workflow Source、Program 与 Run 是分离且独立版本化的 durable artifact。各 Store 只接受当前所属 contract，以 canonical bytes、内容摘要和 revision/generation CAS 约束更新；写入采用同目录临时文件、sync、原子替换和父目录 sync。内存状态只在 durable publish 后更新；rename 已提交但目录 durability 未确认时返回显式 committed-warning，不生成第二个 identity 或伪装失败。

Blob Store 独占 immutable content-addressed bytes，发布前验证 digest/size/quota，引用与租约决定 GC 可达性。Stream 和 Resource lease 只属于 Run 生命周期，不能进入 Workflow Source、Program 或 durable Run value。Run Store 持久化 QUEUED/RUNNING/terminal 状态、NodeAttempt 与 AdapterAction；重启只把遗留 RUNNING 转为 INTERRUPTED，不透明重放副作用。

Node Package Store 使用另一套严格的 registry-last generation 模型：验证后的不可变 generation 先在同一文件系统 durable publish，canonical `registry.json` 最后原子替换并成为唯一安装 authority。registry v2 同时持有 monotonic trust policy、publisher signature evidence、namespace ownership、revocation/quarantine 与 package pointers，避免 trust 和 generation authority 跨文件提交。registry 写入 rename 前失败不发布内存；rename 已提交但目录 durability 未确认时按 `durablefs.Committed` 发布同一代并返回 warning。incoming/orphan generation 没有 authority，reopen 时清理；所有 registry 引用的 generation 都重新验证 manifest、精确文件集合、size、SHA-256、mode、Ed25519 signature 和当前 trust status。
