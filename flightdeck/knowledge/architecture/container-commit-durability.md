---
kind: note
summary: "历史 3.0 Container lock-last 持久化协议；现行 3.1 已删除 Container，仅在旧基线/归档取证时读取。"
activation: action
read_when: "仅在恢复 archived Topic go-backend-architecture-review，或审查 8316d590 附近的 3.0 Container 行为、归档迁移记录时"
recheck_when: "新增权威容器文件；改变 lock schema/hash；引入 import；实现 generation directory/current pointer；改变外部改盘或 MCP 写入契约"
---
# Container lock-last commit 与恢复语义

> 历史知识：3.1 已物理删除 Container Store/runtime，不得把本文协议恢复成现行持久化或兼容层。当前 Workflow/Program/Run/Blob/NodePackage 持久化见 `docs/architecture/storage.md`。
## 权威提交协议

- `Store.Save` 先 deep clone caller，再 validate；持 store writer lock 固定写 `package.json` → `graph.json` → `installation.json` → `yotta-lock.json`。
- 每个文件用同目录 temp、file sync、atomic replace；Unix replace 后 sync directory，Windows 使用 `MoveFileEx(REPLACE_EXISTING | WRITE_THROUGH)`。
- `yotta-lock.json` 是 commit record。schema v2 记录 manifest、graph、installation 与 dependency closure hash；不能把 map iteration 当作写入顺序。
- 任一步失败都按逆序恢复写入前 snapshot。新文件回滚删除也要经过目录 durability barrier；Windows 先 write-through rename 为无权威意义的 tombstone。
- rollback 成功时 cache 不发布；rollback 自身失败时同一进程立即把该容器隔离为 incompatible，不能继续用旧 cache 掩盖磁盘混代。

## 加载与迁移

- load 校验 lock schema、package schema/kind/entryGraph、installation schema、package/installation identity、三个权威文件 hash，以及 `closureHash == hash(dependencies)`。
- 缺 lock、坏 JSON、hash/schema/身份不一致都作为可枚举的 incompatible container 加载，不 panic，也不静默 aggregate 为可运行对象。
- v1 lock 在旧契约下验证后可正常读取；升级到含 installation hash 的 v2 是 best-effort。只读目录或临时写失败不能让有效旧数据不可用，下一次成功 Save 会自然写回 v2。
- lock-last 能检测进程崩溃留下的混合代，但不保证磁盘上自动恢复为完整旧代。若产品要求崩溃后物理上只能存在完整旧/新代，再实施 staging generation + current pointer。

## Snapshot 与导出

- `Get`、`List`、`Reload` 返回 deep snapshot，调用方不能修改 store cache 内的 map/slice。
- Export 在 store read lock 下 deep clone cache；package/graph/lock 各读取一次，验证与写 ZIP 复用同一 raw bytes，避免进程内 Save 或外部改盘在二次读取时制造混代 bundle。
- portable bundle 不包含 `installation.json`，并清空 lock 的 `installationHash`；本机 installation 信息不能泄漏进可移植包。
