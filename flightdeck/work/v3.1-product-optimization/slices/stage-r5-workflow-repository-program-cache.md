# R5 — Workflow Repository 与 Program cache

## Outcome

Workflow Source 的 canonical artifact、metadata、revision CAS、Blob reference 与 quarantine 进入 Content
Catalog；portable Source/Bundle 的 format/version、canonical bytes 和导入导出合同保持不变。Compiler
Program 明确进入 `<cache>/programs`，按 compiler build + Node Catalog identity 分代，并按数量、字节配额与
持久 LRU 自动回收。

正式旧 data-root 的发现、dry-run、backup、resume 和 recovery 仍属于 R7；R5 不在 Store 构造器里偷偷扫描
或搬迁旧 `workspace/workflows`。

## Current

Finished。Content Catalog schema 3 已新增 Workflow Repository；
production GUI/CLI composition 直接注入同一 Catalog Foundation 的 Workflow 与 Object Repository。Source
提交只接受 CAS inventory 中已发布且 active 的完整 Blob 集合，失败时 Source revision 与旧引用一起回滚。

Program cache layout 2 已从 durable workspace 分离，generation identity 同时锁定 compiler build 与 Node
Catalog hash；stale generation、损坏 entry、count/byte overflow 都只清理可重建 cache，不阻止健康 Source
重编译。

## Deep module

对 application/service 保持现有 `SourceStore`、`SourceSnapshot`、`ProgramStore` 调用面：

- `workflowstore` 继续负责 strict Source schema、canonical artifact、revision 语义和 compiler Program trust。
- `storage/catalog.WorkflowRepository` 隐藏 SQLite schema、transaction、reference mirror 和 quarantine。
- composition root 只从 `storage.Roots` 投影 `<cache>/programs`，业务模块不拼接平台目录。
- Source、Bundle、Compiler 和 runtime 仍是唯一产品路径；Catalog 只替换本机 durable representation。

## Persistence

Content schema 3 增加：

- `workflow_sources`：当前 canonical Source bytes、name、revision、hash、format/version 与 Catalog 时间。
- `workflow_refs`：当前 Source 的完整、稳定排序 Blob reference set。
- `workflow_quarantine`：隔离 ID、原名、原因、原始 bytes 与时间。
- 同一事务镜像 `object_refs(owner_kind='workflow-source')`，供 CAS reachability/GC 使用。

Program cache layout 2：

- 根 marker 只声明可重建 cache ownership。
- generation 由 `compiler build digest + Node Catalog digest` 派生。
- Program 文件仍以自身 content hash 命名；mtime 是跨重启 LRU access fact。
- 启动只索引文件名/size/mtime；打开时 strict validate，损坏或过期 entry 删除并按需重编译。

## Commit and recovery

1. Source 先经正式 schema canonicalize，并 inventory 所有 Workflow Resource/direct Blob binding。
2. Catalog transaction 以 expected revision 更新 Source bytes/metadata。
3. transaction 删除旧 `workflow_refs/object_refs`，只从 active `gc_objects` 插入完整新集合。
4. 任一 Blob 缺失、size 冲突或 revision 失配时完整 rollback，旧 Source 与旧引用保持可用。
5. quarantine repair 在同一 transaction 创建 revision 0 Source、恢复 references 并删除隔离记录。
6. Program miss/corruption/stale identity 只返回 cache miss 或重建，不把 derived artifact 当作用户内容故障。

## Verification

- [x] Source create/update/delete/reopen 保持 exact revision/hash CAS 与 canonical artifact。
- [x] Source reference 只接受 active CAS object；失败提交不改变 Source revision 或 GC reachability。
- [x] quarantine list/repair/delete 由 Catalog 持久化并保持单对象隔离。
- [x] 1,000 Source 查询/分页现有真实 service fixture 通过 Catalog-backed SourceStore。
- [x] Program compiler/catalog identity、跨重启 reopen、stale generation 与 corrupt rebuild fixture 通过。
- [x] Program count quota、byte quota 与 persisted LRU eviction fixture 通过。
- [x] 相关 Catalog/Workflow/Application/AppBootstrap/Bundle/Service race 通过。
- [x] `task check`、production build 与隔离 Windows startup smoke 通过。

## Evidence

- Catalog/WorkflowStore/Application/AppBootstrap/Bundle/Workflow Service 定向普通测试与 race 通过；1,000
  Source 查询、CAS 引用回滚、quarantine repair、corrupt cache rebuild、count/byte/LRU eviction 均有 fixture。
- 最终 `task check` 由可续接后台 wrapper 保留同一进程并返回 `EXIT=0`；bindings、AI eval 与 48 个受影响
  Go 包通过。
- `task build` 返回 0，production bundle 预算通过，3.1.0 Windows GUI metadata 正确，隔离 RootSet desktop
  存活 5 秒。
- production CLI 在隔离 profile 初始化后 health 返回 Content Catalog schema 3、Run Ledger schema 1、
  WAL/FULL 与两库 healthy；cache 只出现在 RootSet `cache/programs`。

## Non-goals

- 不改变 `.yotta-workflow`、Workflow Source schema、compiler Program schema 或 Bundle manifest。
- 不把图片、Macro、InputClip payload 或完整 Program bytes放进 Content Catalog。
- 不提前实现 R6 Run Ledger 或 R7 正式旧 data-root migration/recovery UI。
- 不保留 production 的旧文件 Source Store 或 workspace Program durable path。
