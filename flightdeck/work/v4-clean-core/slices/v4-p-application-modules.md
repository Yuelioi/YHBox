# V4-P Application 深 Module

## Goal

保留 `Application` 作为 GUI、CLI、AI、MCP 与 Schedule 的统一命令入口，但把 Source transition、
Run admission 和 Run coordination 的独立决策收进具体深 Module，减少一个 1,400 行对象同时维护多套
状态机和一致性条件。

## Status

Finished

## Completed

- 新增 concrete `sourceTransitions`：统一 patch reduction、节点合同兼容迁移、状态变量迁移安全检查、
  candidate compile、opaque prepare/commit 与 revision/hash CAS。
- `Application` 只在外层维护 command lifecycle lock；现有 patch、prepared artifact 和 state migration
  测试继续通过，没有新增 pass-through interface 或重复测试。
- 新增 concrete `runAdmission`：Program 必须先持久化，再在同一锁内获取 generation lease、复制
  provider snapshot 并执行 admission；execution environment replacement 与它共用同一 Module。
- Application 不再分别持有 `admitter`、`providers`、`providerLease`、`authoringEngine`、Compiler 和
  Blob verifier 等需要成组同步的字段。
- 新增 concrete `runCoordinator`，唯一拥有 queue/jobs、worker context、provider lease 释放、
  Run Owner、debug session、journal 终态与 Run/debug 事件；Application 只保留命令生命周期和跨域协调。
- 新增 concrete `sourceLibrary`，统一搜索预算、筛选、facet、排序、分页和批量 metadata/export/delete
  的逐项结果与引用保护；Wails service 只保留 DTO 投影和单项 RPC。

## Verification

- `go test ./internal/application -count=1`
- Source transition、单 worker Run、热替换 generation lease、取消与真调试测试全部通过。
- 中间切片不运行 `task check`。

## Next

继续 [Compatibility 与无调用表面清理](v4-q-compatibility-deletion.md)。
