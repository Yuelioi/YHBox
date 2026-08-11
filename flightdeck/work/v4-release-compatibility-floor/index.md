# V4 发布兼容性封板

## Goal

以当前生产代码、schema、生成合同和测试为事实源，审计并补齐产品、节点、工作流及其他持久化版本域的
升级流程；冻结 V4 首发兼容底线，确保后续升级不会让已保存的本地工作流静默消失或整体失效。

## Status

Open

## Current

仓库内兼容性封板已经完成并通过发布级全量门禁。V4 的版本域、built-in NodeRef、TypeRef 与 CapabilityRef
均已有不可覆盖的 4.0.0 release floor；Workflow、Bundle、Snippet、Schedule、Macro 等迁移路径已补齐持久写回、
重开或可诊断失败测试，资产列表不再吞掉底层错误。Wails Go/CLI 已统一升级到 `v3.0.0-beta.6`，配套
`@wailsio/runtime` 为上游随该版本发布的 `3.0.0-beta.5`；本地正式 Windows build 与全量门禁均通过。

canonical 源码仓库已确定为 `github.com/yuelioi/yotta`，本地 `origin` 已从旧 YHBox 切换并通过 SSH
读取验证；About、README 与安全报告入口也已同步。4.0 实现基线已提交为精简主线上的 `c4f86a7b`；公开
`main` 从 `e330f47b`（V4 workflow core cutover）的等价无父提交开始，在实现基线处共 17 个提交。完整旧历史保存在
`archive/pre-v4-full-history`，tip 为 `5ac24dee`；两条分支的最终文件树一致并已推送核对。

本地 `main` 与 `origin/main` 已同步且工作树 clean。发布当前只剩从 clean worktree 生成 frozen candidate、
执行 smoke，以及签名和公开发布外部前置项。

## Next

1. 从 clean worktree 按 [发布说明](../../../RELEASING.md) 执行 `task package` 与 frozen candidate smoke。
2. 处理签名、ruleset、release environment 等公开发布前置项。

## Progress

- 2026-08-11 建立发布兼容性封板 Work；确认不把文档或 Knowledge 当作事实源，也不在逐节点/逐操作热路径
  恢复已删除的安全验证层。
- 2026-08-11 冻结 4.0.0 的 39 个公开版本域、147 个 NodeRef、24 个 TypeRef 与 5 个 CapabilityRef，并把
  compatibility check/release 要求接入 Task、CI 与 package。
- 2026-08-11 补齐 Workflow Source/Bundle/Snippet NodeRef、Schedule v1→v5、Macro v1→v2 的 durable migration；
  List/Get 解码与 Catalog 错误改为明确返回或可见 warning。
- 2026-08-11 `task check`、`task check:full`、durable store race、`task smoke:storage-migration` 和两个
  `*:compatibility:release` 均通过；修正 storage smoke 的过期 schema 7 断言及并行计时测试抖动。
- 2026-08-11 将 Wails Go 模块、CLI、前端 runtime、CI、README 与工具链 manifest 统一升级至
  `v3.0.0-beta.6` / `3.0.0-beta.5`；`task build`、桌面启动 smoke 与升级后的 `task check:full` 均通过。
- 2026-08-11 将 canonical 源码入口与本地 `origin` 切换到 `Yuelioi/yotta`；以原创的 Y 形工作流节点标识
  替换旧 Pixiv 图，完成 SVG/PNG/六尺寸 ICO、标题栏和 About 投影。`task build` 与 `task check` 通过。
- 2026-08-12 重构 About 为产品能力导向页面，保留作者与主页入口并移除技术栈陈列；普通节点和子图节点改为整块
  非交互表面可拖动，按钮、输入与端口继续通过 `nodrag` 隔离。定向测试、前端生产构建、`task check`
  与正式 Windows `task build` 均通过。
- 2026-08-12 将完整旧主线发布到 `archive/pre-v4-full-history`；以 `e330f47b` 的等价文件树建立无父根并
  重放后续 V4 提交，将公开 `main` 在实现基线处精简为 17 个提交。归档/精简 tip 与最终树均在远端核对一致。

## References

- [稳定上下文](context.md) — 本轮兼容底线、约束和验收定义。
- [既有版本域解耦 Work](../version-domain-decoupling/index.md) — 前一阶段的独立版本域设计与 v1 cutover。
