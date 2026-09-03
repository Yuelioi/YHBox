# 错误契约系统整改

## Goal

让 Yotta 的 RPC、异步事件和持久 Run 失败都沿 canonical Problem 完整投影，核心用户路径不再退化为
`transport.unstructured_failure`、`UNKNOWN_ERROR` 或无恢复动作的原始字符串；真实 transport 破坏必须被自动化
门禁捕获并留下可关联证据。

## Status

Finished

## Current

P0 service 注册断点已修复：16 个 Wails service 全部通过 `NewServiceWithOptions` 显式安装 `apperr.Marshal`，静态
注册门禁要求恰好 16 个 canonical 注册且禁止退回 `NewService`。另有真实 Wails `Bindings.Add → BoundMethod.Call
→ CallError.Cause` 测试断言 stable ID、category 与 operation ID；此前隔离 WebView 探针已覆盖 Settings、Hotkey、
Workflow 三类实际失败。

Workflow authoring 不再把 candidate diagnostics 压成 `INVALID_RESULT`：`PatchError` 现保留首个 diagnostic 的
code、graph path、node ID 与 field path，保存 UI 能从 canonical params 定位旧节点。Hotkey conflict/reserved/invalid
已有 stable validation Problem，Pause/Resume/Update 未分类失败也有领域 ID。

Hotkey manifest 的用户可见失败已改为 canonical `problem`；原始 `LastError` 不再序列化或显示。冲突、保留、
非法格式、OS 注册、更新、暂停、恢复和回滚都有稳定 ID 与恢复文案。

Settings、Calibration、Schedule 已补齐 RPC boundary 的稳定 Problem，并明确区分设置/计划的未提交失败与
已提交但 live sync/reload 失败。InputClip、Macro 的 CRUD 主路径也已完成 validation/domain/infrastructure
分类；Recording 与窗口捕获前端已删除 `payload.error` 兼容旁路。transport fallback 现在仅记录脱敏后的
operation 与 shape，不再让真实契约破坏悄无声息。

整改已完成。Snippet、Workflow、Tools、Asset 和 AI 凭据/评估边界已完成用户可见映射。工作流保存现在先检查内存草稿，
阻断诊断可定位 graph/node/field 并保留未保存输入。Run Journal 与 AI Conversation 的 durable Problem identity
已核验；静默 catch 仅保留在明确的幂等窗口清理或可选探测路径。

## Next

无。

## References

- [稳定上下文](context.md)
- [阶段计划](plan.md)
- [错误契约开发指南](../../knowledge/errors/error-contract.md)

## Progress

- 2026-09-04 建立 Work；真实 WebView 探针确认全局 `application.Options.MarshalError` 被 service-level 默认
  marshaler 绕过，Settings/Hotkey/Workflow 三类错误均稳定复现为 `transport.unstructured_failure`。
- 2026-09-04 修复全部 service-level marshaler 并增加 Wails 框架级门禁；Workflow 保存保留节点定位诊断；Hotkey
  validation 错误开始迁移到 stable Problem。
- 2026-09-04 完成 Hotkey 用户可见 Problem 收敛，移除前端 raw `LastError`；增量 `task check` 全部通过。
- 2026-09-04 完成 Settings、Calibration、Schedule 及 InputClip/Macro 主路径映射；移除 Recording/窗口捕获
  `payload.error` 前端旁路，并为 unstructured fallback 增加脱敏 transport shape 报告。
- 2026-09-04 完成 Snippet 用户可见失败映射；对应包测试与 3093-key i18n parity/compile 检查通过。
- 2026-09-04 完成 Workflow、Tools、Asset、AI 凭据/评估映射和保存前草稿诊断；新增 service Problem 翻译
  完整性及 legacy event bypass 门禁。`task check` 与真实隔离 `task webview:smoke` 通过。
- 2026-09-04 完成最终验收：增量门禁通过；真实 WebView journey 通过；全仓覆盖率达到 65.0%。`task check:full`
  的其余阶段仅被未改动文件中的既有 staticcheck baseline 阻断（16 个 ST1005、2 个 U1000）。
- 2026-09-04 经 owner 授权顺带清理实际 19 项 staticcheck baseline（17 个 ST1005、2 个 U1000）；最终
  `task check:full` 全部通过，全仓覆盖率 65.1%。
