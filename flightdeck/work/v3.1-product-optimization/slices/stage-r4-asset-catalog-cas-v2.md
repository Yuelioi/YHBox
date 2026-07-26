# R4 — Asset Catalog + CAS v2

## Outcome

Global Asset 的小型可查询元数据、variant、tag 与 Blob 引用进入 Content Catalog；不可变大字节继续保留在
文件 CAS。工作流资源和全局资产仍是两个领域对象，只共享精确 `BlobRef` 指向的物理字节。

当前开发期文件型 Asset Store 与平铺 CAS 不兼容读取、不自动搬迁；正式发布后的变更继续通过独立 schema/
layout version、不可变 migration registry、backup 和 recovery 流程升级。

## Current

Finished。Global Asset 已由 Content Catalog schema 2 持久化，CAS layout 2 使用两级分片、Catalog inventory、
运行期 pin、持久 lease、宽限 GC 与可恢复 trash。Catalog 只接受已经由 CAS inventory 观察且仍为 active 的对象，
提交失败只会留下可宽限回收的孤儿字节。

## Deep module

外部接口保持用户已经使用的 `asset.Service` Wails 动作。其下的持久化模块隐藏：

- SQLite schema、SQL、全局/记录 revision、分页/筛选/facet 查询和事务。
- Blob staging、流式 digest、分片物理路径、dedup、quota、runtime pin 和 trash。
- `CAS publish → Catalog metadata/object_refs commit` 顺序以及失败后孤儿对象的宽限回收。
- mark/grace/recheck/trash/sweep GC 状态机。

不新增泛型 `Store[T]` 或向 service 暴露 `*sql.DB`。只有一个 SQLite adapter，不为测试制造平行 Repository
port；测试通过同一真实临时 SQLite/CAS 模块接口。

## Persistence

Content schema 2 增加：

- `assets`、`asset_variants`、`asset_tags`：Global Asset 元数据与服务端查询索引。
- `object_refs`：`owner_kind + owner_id + role -> BlobRef` 的权威引用。
- `gc_objects`：物理 generation、observed size、state、`unreachable_since` 和错误/重试事实。
- `object_leases`：需要跨进程/恢复存活的有期限 pin；单 Run reader/writer pin 仍可在内存中。

CAS layout 2 使用 `<objects>/ab/cd/<64-hex-digest>`；staging 和 trash 使用专用内部目录，启动只恢复这些
有界状态，不再扫描/哈希全部对象计算 quota。物理路径永远不进入 `BlobRef`、Workflow Source 或 Bundle。

## Commit and recovery

1. 在 CAS staging 流式写入、限额、hash、sync。
2. 原子发布 immutable object；dedup 时验证已有对象。
3. Content transaction upsert asset metadata、variants/tags、`gc_objects` 与完整 `object_refs`，递增 revision。
4. 第 3 步失败只留下不可达对象；不能留下指向缺失字节的已提交引用。
5. GC 首轮只 mark `unreachable_since`；越过宽限期后重新核对引用和 pin，先移入 trash，再 finalize Catalog。
6. kill-point 后由 staging/trash/GC state 恢复；不得把“可能已提交”伪装成可盲重试的失败。

## Verification

- [x] 10,000 Asset metadata 的分页、搜索、kind/category/tag、facet 和稳定排序不全量 materialize。
- [x] 100,000 Blob 的启动、配额和 GC 不依赖根目录全量扫描或全量 hash。
- [x] 同 digest dedup，冲突 size/bytes fail closed；256 MiB 路径保持流式。
- [x] Asset/variant/blob commit、delete、并发更新和 revision 语义保持现有 Wails 行为。
- [x] Workflow Resource 与 Global Asset 可共享 bytes，删除 Asset 不破坏仍被工作流引用的对象。
- [x] GC preview/commit 具备宽限期、stale token、runtime pin、二次引用检查和 trash recovery。
- [x] staging publish、Catalog commit、mark、trash、finalize kill-point 均有恢复测试。
- [x] `task check`、production build 和隔离 Windows startup/asset smoke 通过。

## Evidence

- 10k Asset/100k object inventory、Catalog/CAS reopen、缺失引用拒绝、并发 variant、宽限/lease/pin/stale
  preview 与 staging/trash/Catalog failure fixture 通过；相关 Blob/Catalog/Asset race 通过。
- `task check` 按 30 个变更文件运行 bindings 与 48 个受影响 Go 包，最终同一进程退出 0。
- `task build` 生成 3.1.0 Windows GUI/CLI/worker，binary metadata 与隔离 5 秒 desktop startup smoke 通过。
- `task webview:smoke` 在显式 owned 的隔离 RootSet 上退出 0；资产库截图已目检，无 JS/console 错误。

## Non-goals

- 不在 R4 迁 Workflow Source/Program/Run durable representation。
- 不把图片、Macro、InputClip payload 放进 SQLite BLOB。
- 不实现开发期旧 JSON/平铺 CAS compatibility reader。
- 不把 Asset GUID 写入 Workflow Source 代替 `BlobRef`。
