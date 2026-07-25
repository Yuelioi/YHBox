# R1 — 配置与资源持久化审计

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

1. 当前持久化清单与容量/一致性风险矩阵。
2. 目标目录树和配置/资源领域所有权决策。
3. Storage/Repository、索引、原子写、锁、备份恢复、GC 与 migration seam 方案。
4. 后续实现 Slice、兼容边界、自动化门禁和真实大数据 fixture 验收计划。

## Next

从仓库入口、`internal/services/`、现有 store、data-root 初始化和 schema 开始只读审计；使用 1000+
Workflow/Asset、GB 级 Blob 和崩溃中断场景校准目标，而不是只为当前小样本设计。
