# Own durable Run facts and ephemeral authority separately

Yotta 3.1 将一次执行拆成 durable `RunRecord` 与 ephemeral `Run Owner`。RunRecord 通过 generation/digest CAS 持久化 Program、Catalog、Capability Plan、Run Grant、policy generation、状态、稳定错误与 durable Run Value；Run Owner 持有 cancellable context、Grant Authorizer、Resource Broker 和 provider objects。两者共享 Run ID，但 RunRecord 不保存 stream/resource token、provider object、credential secret 或 bearer authority。

状态只允许 `QUEUED → RUNNING → SUCCEEDED|FAILED|CANCELLED|INTERRUPTED` 和 `QUEUED → CANCELLED`。写入 QUEUED 成功后才可通知 Worker；进程重启将遗留 RUNNING 原子写为 INTERRUPTED，不自动重放未知 effect。终止时先 revoke Grant，再 cancel Run context、revoke leases 并永久 close Broker。Provider open 收到的 canonical capability scope 与 credential binding metadata 由 Authorizer 返回并由 Broker 注入，workflow config 不能伪造授权范围。通用 borrow 只允许 canonical scope 完全相同并继续收窄 operation/expiry；跨 scope delegation 必须由 provider 显式建模，Broker 不猜测 JSON scope 的子集关系。

我们拒绝让 mutable workflow/container state 充当运行记录、拒绝把内存状态作为磁盘损坏时的 fallback，也拒绝让 builtin/GUI 绕过 admission。NodeAttempt/AdapterAction journal、Target Planner 与 production interpreter 在后续纵向切片接入同一模型；接入前 preview 继续 fail closed。
