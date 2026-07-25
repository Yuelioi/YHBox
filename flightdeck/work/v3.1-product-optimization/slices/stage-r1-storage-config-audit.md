# R1 — 配置与资源持久化审计

## Status

Completed — 2026-07-25

## Journey

Yotta 已同时持久化应用设置、工作流 Source/Program、Global Asset、Workflow Resource、CAS Blob、录制、
计划、运行记录、插件/节点包和本机目标配置。当前功能已经进入中大型桌面软件的规模，但数据根目录、
配置所有权、索引、容量边界、并发写入、恢复与迁移策略尚未形成一份统一且可机械验证的架构。

本 Slice 先建立事实和目标模型，不直接搬目录：

- 枚举全部持久化对象、当前路径、读写者、schema/version、生命周期、敏感级别和预期数量/体积。
- 区分用户配置、安装级状态、工作区内容、不可变内容寻址对象、可重建索引/缓存、运行历史与日志。
- 评估大数据量下的目录扫描、单 JSON 文件、全量反序列化、索引缺失、锁/原子写、备份恢复和 GC 风险。
- 给出目标 data-root 布局、存储接口、索引边界、事务/锁策略、配额与清理、完整性检查和 migration registry。
- 把改造拆成可独立上线、可回滚且保持现有 Source/compiler/runtime 事实不变的后续 Slice。

## Constraints

- 开发期不为已废弃格式编写一次性兼容 reader，但正式存储层必须内置显式 schema version、migration registry、
  dry-run/backup/commit/recovery seam，为上线后的逐版本迁移负责。
- Secret、credential、本机 consent 和精确 target installation 身份不能混入可移植 Workflow Source、Global
  Asset 或内容寻址 Blob。
- 大对象不进入通用 settings JSON；可重建索引/缩略图/缓存不冒充唯一事实；CAS 字节与 metadata、引用和
  生命周期分层。
- 不以一次“大搬家”同时重写所有 service。先建立边界和验证工具，再按对象族纵向迁移。

## Deliverable

1. [当前持久化清单与容量/一致性风险矩阵](../references/storage-config-current-audit.md)。
2. [目标目录、领域所有权与深模块方案](../references/storage-config-target-architecture.md)。
3. [外部一手资料研究](../references/storage-config-primary-research.md)。
4. 后续实现从 [R2 — RootSet 与持久化生命周期基座](stage-r2-storage-root-lifecycle.md) 开始。

## Next

实现 R2 的平台 RootSet、显式开发/便携 override、settings recovery envelope、统一 diagnostics 路径、
根级 writer lease 和完整 version inventory；在 migration engine 就绪前不自动搬动或删除现有数据。

## Result

- 当前不到 1 MiB 的开发数据已同时分散在 exe 同目录、`data/` 和相对 `logs/`，并由多套启动扫描、
  原子写、损坏和版本策略管理；Run journal 还存在累计 O(n²) 重写风险。
- 目标不是“全部 SQLite”，而是 Content Catalog + 独立 Run Ledger + 文件 CAS + portable artifacts；
  settings 保持小型 recovery envelope，secret 继续由平台 secure store 管理。
- 产品版本、artifact schema、DB schema 和物理 layout 分离。正式迁移必须有注册 step checksum、
  preflight/dry-run、backup、apply、verify、commit 和 resume/recovery。
