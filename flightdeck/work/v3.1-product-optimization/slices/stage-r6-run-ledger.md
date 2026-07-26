# R6 — Run Ledger

## Outcome

Run history 从 `workspace/runs/<run-id>.json` 迁入独立 `state/runs.db`：Run summary、append-only event 和
terminal value/blob reference 分表持久化。现有 `run.Record` generation/digest、状态机、journal 顺序与
redaction 事实保持不变；SQLite Repository 只替换 durable representation。

## Current

Finished。Run Ledger schema 2 提供 `runs`、`run_events`、`run_values` 与查询/retention 索引。
`run.Store` 是 domain Interface，继续验证 canonical Record、exact previous digest 和合法 successor；
`storage/catalog.RunRepository` 是唯一 SQLite Adapter，原子提交小型 Run head 与单个 event，terminal
transition 才更新 summary/value。

production GUI/CLI composition 从同一 Catalog Foundation 注入 Run Repository。旧
`workspace/runs` v1 目录会按 ownership marker 幂等导入，旧 JSON 原地保留为 rollback data；正式
data-root dry-run、snapshot、resume/rollback 和 recovery UI 仍属于 R7。

## Deep module

- `internal/run` 保持 Run domain Module：Record seal/digest、状态机、journal order、redaction、value trust。
- `storage/catalog.RunRepository` 隐藏 SQLite schema、snapshot、CAS update、分页与 retention transaction。
- event append 只更新 generation/digest/journal count 并插入一个 immutable event，不重写 summary/value。
- 完整 Record reopen 在一个只读 SQLite snapshot 中组合 summary/events/values，避免并发 append 产生混合
  generation；timeline page 则只读取 bounded summary 与一页 event。
- application 的 durable Blob inventory 直接查询 `run_values` payload refs，不再打开全部 Run Record。

## Persistence

Run schema 2 增加：

- `runs`：当前 generation/digest/status/timestamps、bounded summary artifact、journal count、archive fact。
- `run_events`：`run_id + sequence` append log、kind、occurred time 与 canonical event artifact。
- `run_values`：terminal value identity/digest/artifact，以及可选 CAS media type/digest/size。
- status/queued、archive/ended、event time 与 payload digest 索引。

Run summary artifact 不包含 generation/digest、journal 或 values；事件追加时因此保持不变。完整 Record
仍由原 domain schema seal 并核对最终 digest，Ledger 不能创造新的状态转换。

## Query, retention and migration

1. timeline page 在同一 SQLite read snapshot 读取 summary 与 `ORDER BY sequence DESC LIMIT/OFFSET`，
   对页面内事件恢复正序；默认 200、最大 500。
2. archive 只标记 bounded terminal Run，历史和 payload roots 仍可查询。
3. purge 只删除已经 archive 且早于 cutoff 的 Run，并由 foreign-key cascade 清理 event/value。
4. Blob GC 直接从 `run_values` 返回去重 payload CAS refs。
5. legacy import strict reopen 每个 canonical JSON，并按完整 digest 幂等写入 Ledger；任一冲突/损坏
   fail closed，不删除旧文件。

## Verification

- [x] Run schema 1 → 2 migration、application identity、required objects 与 health 通过。
- [x] queued/running/event/terminal/value reopen 保持原 Record generation 与 digest。
- [x] stale generation、第二 journal owner、out-of-band Ledger mutation fail closed。
- [x] concurrent append/read 使用一致 snapshot；Catalog/Run targeted race 通过。
- [x] timeline page、archive、purge 与 payload Blob inventory fixture 通过。
- [x] legacy one-JSON-per-Run import 幂等，旧 bytes 保留。
- [x] 全 `internal/...` 普通测试通过。
- [x] `task check`、production build、Windows metadata/隔离 startup 与 production CLI health 通过。

## Evidence

- 全 `internal/...` 测试退出 0；`go test -race ./internal/storage/catalog ./internal/run` 退出 0。
- 最终 `task check` 在同一可续接执行单元中退出 0：AI eval 8/8、Wails contract 与 30 个受影响 Go 包通过。
- `task build` 退出 0；bundle budgets、3.1.0 Windows GUI metadata 与隔离 5 秒 startup smoke 通过。
- production `Yotta.CLI.exe` 在隔离 profile 初始化后 health 返回 Content schema 3、Run schema 2、
  WAL/FULL、quick-check ok 与两库 healthy。

## Non-goals

- 不改变 Run Record format/version、domain 状态机、journal fact 或 runtime execution path。
- 不把大 payload bytes 放进 runs.db；数据库只保存 value artifact 与 CAS reference。
- 不删除 legacy Run JSON，也不提前实现 R7 的跨版本 snapshot/resume/rollback/recovery UI。
- 不把 Run 与 Content Catalog 合并为一个事务域。
