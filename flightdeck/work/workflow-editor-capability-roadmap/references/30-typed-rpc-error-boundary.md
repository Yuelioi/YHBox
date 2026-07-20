# Slice 30：Typed RPC 错误边界

## Outcome / Question

建立唯一 RPC 错误契约，消除 transport 吞错、自动 toast、`undefined` 伪成功和随后生成的二次假错误，让每个 domain action 能按稳定 code 决定原地恢复方式。

## Completion criterion

- backend error envelope 至少包含 code、category、message、details、operation/run ID 和 retryability。
- Wails transport 只 decode、关联上下文并 rethrow；禁止自动 toast 或返回伪成功值。
- domain action 明确选择 inline、字段错误、确认 modal、恢复动作或 failure toast。
- 保存/安装等原地成功不 toast；代码中无 browser alert/confirm/prompt。
- 录制 finalize 等链路只显示一个权威失败，不追加 invalid-result 假错误。

## Blocked by

Slice 29。

## Verification

- contract/schema tests 覆盖 Go → Wails → TypeScript 错误字段与 unknown fallback。
- 前端测试覆盖 transport reject、domain recovery 与单一反馈。
- 全仓静态扫描禁止 browser alert 和通用 RPC success/error toast。
- Stage R1 结束时批量执行 G11、相关聚合测试与 `task check`。

## Out of scope

- 不在此 Slice 修复每个 domain 的业务根因。
- 不把后端内部堆栈或敏感 target/credential 信息暴露给前端。
- 不为成功动作增加新的全局通知系统。

## Result

Completed。

- `application.Options.MarshalError` 现在一次性把所有 Go service error 投影为稳定 envelope；typed domain errors 可通过窄 `EnvelopeProvider` 暴露安全 facts。
- envelope 包含 code、category、locale-free message、safe details、operationId、可选 runId 和 retryable；Admission error 已接入 typed projection。
- 前端 `invoke`/`callRPC` 只 normalize + rethrow `RPCError`；删除 `invokeVoid`、自动 toast 和 `undefined/false` failure sentinel。
- generated service bindings 只允许由 `lib/backend.ts` 与 workflow transport 直接导入，workflow 路径也已接入 typed seam。
- Settings、AI、Schedule、Hotkey、Application/Automation、Launcher、Screen Picker 等已把失败决策放回 domain action；保存成功仍使用原地状态。
- 这同时修复了两个系统性假象：recording 原始失败后不再继续校验 `undefined` 产生 `invalid result`；void calibration Start 成功不再被当作失败。
- 定向证据：Go `apperr/admission/desktopapp` tests、frontend typecheck、5 个相关 Vitest 文件共 41 tests、`git diff --check` 通过。R1 完整 `task check` 仍按阶段规则在 Slices 31–33 完成后统一执行。
