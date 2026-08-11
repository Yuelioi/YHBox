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
读取验证；About、README 与安全报告入口也已同步。发布当前只剩 Git 状态阻塞：工作树仍有约 200 项状态
记录尚未 review/commit，因此不能运行要求 clean worktree 的 `task package`。公开主线的历史切点已选定为
`e330f47b`（V4 workflow core cutover）；旧历史将完整保存在 `archive/pre-v4-full-history`，新 `main` 只重放
该切点及其后的 V4 提交。

## Next

1. Commit 当前工作树；保持三份 4.0.0 compatibility snapshot、品牌资源与实现同批进入历史。
2. 创建旧历史归档分支，重建并推送精简的 V4 `main`，核对远端提交计数与树内容。
3. 从 clean worktree 按 [发布说明](../../../RELEASING.md) 执行 `task package`，再处理签名和公开发布前置项。

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

## References

- [稳定上下文](context.md) — 本轮兼容底线、约束和验收定义。
- [既有版本域解耦 Work](../version-domain-decoupling/index.md) — 前一阶段的独立版本域设计与 v1 cutover。
