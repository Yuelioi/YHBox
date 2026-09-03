# 错误契约系统整改上下文

## Product contract

- 用户错误必须说明失败对象或阶段、真实恢复动作和 operation ID；已知 domain/validation 错误不能显示为 transport
  或 unknown。
- `transport.unstructured_failure` 只表示 transport 或调用代码破坏 canonical envelope，不是业务错误类别；核心
  产品验收中出现一次即失败。
- Workflow 编译诊断必须保留 code、graph path、node ID、field path 和修复信息，保存失败不得压扁为
  `INVALID_RESULT`。
- 异步事件只发送 `problem`；Run/Conversation 等长任务继续保存稳定 Problem 与 durable identity。
- 原始 adapter、OS、provider、路径和私有内容只进入进程内诊断日志，不进入用户文案或 envelope params。

## Verified root cause

- Wails v3 beta.6 `Bindings.Add` 为 service 单独选择 marshaler；`application.NewService` 的 options 为空时直接使用
  Wails `defaultMarshalError`，不会继承 `application.Options.MarshalError`。
- Yotta 当前 16 个 service 全部使用 `application.NewService`，因此 150 个 RPC 均未使用预期的 `apperr.Marshal`。
- 真实 WebView 中 Settings 非法值和 Hotkey missing-key 的 `cause` 是空对象；Workflow invalid patch 的 cause 是
  raw `PatchError`，三者都无法被 `normalizeError` 识别。

## Constraints

- 不把原始 message 字符串重新作为协议；修复必须发生在 service registration、domain mapping 或 canonical event
  seam。
- 不用 `system.unexpected` 或新的泛化 ID 掩盖已知语义。
- 保留用户草稿和持久提交边界；错误整改不得改变 Workflow 执行热路径或恢复已删除的 authority 层。
- 每阶段先有能稳定复现原缺陷的失败测试，再修改生产代码。

## Acceptance

- 真实 Wails native/dev 探针覆盖 validation、domain、adapter、infrastructure，全部得到 canonical envelope。
- 核心 UI 与自动化 smoke 中不存在 `transport.unstructured_failure`、`UNKNOWN_ERROR` 或 raw `lastError`。
- Workflow 旧节点、revision conflict、配置无效、持久化失败均显示具体原因并可定位或恢复。
- 所有 Wails service error 出口有 inventory；未完成映射必须显式列入有限 allowlist，并由 CI 阻止扩大。

