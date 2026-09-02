# Error contract audit

审计日期：2026-09-02。范围：当前生产代码与相邻测试；重点为 Node Adapter / Runtime、Installed Automation、Application、Services、Frontend/i18n 与 MCP。

## 结论

当前并非“完全没有错误规范”，而是存在三套成熟度不同、尚未贯通的契约：

1. Wails RPC 已有较好的统一信封：`code + category + message + details + operationId + runId + retryable`（`internal/apperr/apperr.go:24-36`），前端也有统一解码与 i18n lookup（`frontend/src/lib/invoke.ts:17-38,101-122`）。
2. Run/Node 已有稳定 `ErrorCode`、错误类别、节点定位和 attempt，但没有结构化参数或安全诊断信息；Go `Cause` 只存在于执行期间，落盘时被丢弃（`internal/nodeadapter/abi.go:23-42`；`internal/workflow/compiler/scheduler.go:630-641`；`internal/run/record.go:66-70`）。
3. Automation health、若干 service event/result 和 MCP tool error 仍返回原始英文字符串或字符串拼接的伪 code，不能稳定翻译，也会泄漏 adapter/package/OS 实现细节。

因此用户看到 `adapter xxx` 并非偶发显示 bug，而是错误在生产边界缺少统一的“稳定 ID + 结构化参数 + 仅供日志的 cause”分层。

## 已有的可靠基础

### RPC envelope

- `apperr.Envelope` 明确定义 locale-free transport 字段，并声明 `Details` 不得包含凭据或 ambient host authority（`internal/apperr/apperr.go:24-36`）。
- typed domain error 可通过 `EnvelopeProvider` 投影安全字段（`internal/apperr/apperr.go:38-42`）。Admission 已正确采用这个 seam，并将 `graphId/nodeId/requirementId/slot/commit` 结构化，而不是拼进 message（`internal/admission/admission.go:33-71`）。
- 前端 `RPCError` 保留 code/category/details/operationId/runId/retryable（`frontend/src/lib/invoke.ts:17-38`），并优先用 `error.<code>` 翻译（`frontend/src/lib/invoke.ts:101-122`）。
- i18n 检查会强制当前节点投影声明的所有 error code 都有中英文消息（`frontend/src/i18n/check.cjs:248-286`）。这是节点 error ID 可翻译性的有效机械门禁。

### Run code 与定位

- Node failure 已有稳定 `Code` 和错误输出 port（`internal/nodeadapter/abi.go:23-42`）。Scheduler 校验 code 必须由 Node Contract 声明、error output 必须存在（`internal/workflow/compiler/scheduler.go:611-624`）。
- attempt/action/status journal 都保存稳定 code；Run terminal failure 保存 category、graph/node/attempt（`internal/run/journal.go:149-169`；`internal/run/record.go:66-70`）。
- Workflow service 投影上述定位信息和 timeline code（`internal/services/workflow/service.go:199-228,595-599,653-665`）。

这些能力适合作为统一错误契约的基座，不应另起一套 AI 专用错误格式。

## 关键缺口

### P0：Run 会永久丢失错误参数与 cause

`NodeFailure` 只有 `Code/Output/Cause`，没有 params（`internal/nodeadapter/abi.go:23-42`）。Scheduler 写入失败 attempt 时只落 `ErrorCode + RedactedSummary`，不会把 Cause 或参数持久化（`internal/workflow/compiler/scheduler.go:630-641`）。最终 `RunError` 也只有 code/category/retryable/graph/node/attempt（`internal/run/record.go:66-70`），Workflow API 的 `FailureView` 同样没有 params 或 evidence（`internal/services/workflow/service.go:199-206`）。

后果：

- UI 只能翻译一个粗粒度 code，无法说“哪个 target slot”“阈值是多少”“实际状态码是什么”。
- 开发者只能靠进程日志或复现恢复 cause；历史 Run 本身不足以诊断。
- AI 即使拿到 Run 也无法区分同一 code 下的具体根因。

建议扩展一个有界、可校验、可脱敏的 `FailureDetails`/`ErrorParams`，贯通 `NodeFailure -> Journal NodeAttempt -> RunError/Run Evidence -> Workflow API/MCP`。参数键应由 Node Contract/error declaration 或注册表声明，不接受任意 map 直接穿透；底层 `Cause` 只写关联日志，不能作为用户 message。

### P0：未分类 Go error 会跨 Wails 泄漏原始实现文本

`apperr.From` 对没有实现 `EnvelopeProvider` 的任何错误统一标成 `rpc.unclassified`，但把 `err.Error()` 原样放进 `Message`（`internal/apperr/apperr.go:80-95`）。前端对 `rpc.unclassified` 会调用 `friendlyRawErrorMessage`，除少量 timeout/connection 英文 substring 外原样返回（`frontend/src/lib/invoke.ts:115-148`）。

这意味着诸如 `automation adapter ...`、包名、Win32/API 名、路径、英文 validation 文本会直接呈现给用户。它也使翻译依赖英文字符串内容。正确策略应是：生产 RPC 未分类错误对用户固定返回安全的 `rpc.unclassified` 本地化消息和 `operationId`；原始 cause 进入结构化日志，用 operationId 关联。只有显式 allowlisted safe diagnostic 才能进入 details。

### P0：Automation health 是原始字符串旁路

`AutomationTargetHealth` 公开 `Code` 和 `Message`（`internal/services/automation_service.go:18-22`），但多个分支直接返回 `err.Error()`（同文件 `61-70,84-90`），成功和 not-found 文案也硬编码英文（`79-81`）。Installed automation 的 `Failure.Error()` 明确拼接 `code + ": " + Cause.Error()`（`internal/automation/installed/provider.go:115-127`），所以 adapter、OS、package 细节自然进入设置 UI。

该返回值应改为 `code + params + retryable`，或者复用同一个 locale-free envelope；`Message` 不应成为产品展示契约。Installed `Failure` 可继续携带 Cause，但必须增加安全参数投影，不由 `Error()` 充当 transport。

### P1：Installed automation code 粒度不足

Installed 层有稳定 code 常量（`internal/automation/installed/provider.go:61-70`），但 `automation.window_failed`、`automation.capture_failed`、`automation.input_failed` 等覆盖大量不同用户动作。具体根因只在 Cause 中。例如 registry 对 host 不支持返回 `automation.unsupported_host`，但 Cause 带 adapter/target kind（`internal/automation/installed/adapter.go:293-300`）。

需要区分“用户可行动原因”和“实现 cause”：如 target process missing、window selector no match、device unauthorized、capture backend unavailable、old package/ABI incompatible。不要无限增加叶子 code；可使用稳定 code + 枚举 `reason` + 参数（slot/operation/expectedVersion/actualVersion），其中枚举同样受 schema/测试约束。

### P1：前端仍有字符串启发式和原始 Error.message 展示

- `friendlyRawErrorMessage` 用英文 substring 判断 deadline/connection（`frontend/src/lib/invoke.ts:125-148`）。这是遗留兼容，只应作为 transport 崩坏兜底，不能成为新错误契约。
- 多个编辑器组件直接显示 `error.message`/`String(error)`，例如 AI review（`frontend/src/app/editor/AIWorkflowReviewPanel.vue:351`）、连接编辑（`frontend/src/views/WorkflowEditorView.vue:3513`）、若干 value editor。它们可能绕过 `errorMessage()`。
- Timeline failure 本地化只查 `error.<code>`，未知 code 直接显示 raw code；category/node ID 也直接显示（`frontend/src/app/editor/RunTimelinePanel.vue:79-83,223-228`）。这对开发者有用，但用户缺少行动建议、参数和 operationId。

建议门禁：所有 Wails catch 必须进入 `toRPCError/errorMessage`；所有 Run error/status 进入统一 renderer。raw code、node ID、operationId 放“技术详情”，主文案使用 i18n。

### P1：MCP 错误不是结构化 tool result

MCP input/output schema 是严格、结构化且关闭 additional properties（`internal/services/mcpserver/server.go:121-178`），但 handler error 仍是普通 Go error。典型例子 `catalog_describe` 返回 `fmt.Errorf("UNKNOWN_NODE_TYPE: %s", ...)`（`internal/services/mcpserver/catalog.go:81-85`）：code 和参数被编码进英文字符串。MCP runtime 启动错误也直接包装 socket 文本（`internal/services/mcpserver/runtime.go:53-66`）。

建议定义 MCP 统一错误 data：`code/category/params/retryable/operationId`。业务上可预期的失败（unknown node、revision conflict、compile rejected）宜作为结构化 tool error/result；基础设施错误仍需 operationId，原 cause 仅日志。未来 Run tools 必须直接返回稳定 Run Evidence，而非向模型倾倒 `err.Error()` 或整份日志。

### P2：Services 中还有多个原始 error result/event

可见实例包括 AI profile test `Error: err.Error()`（`internal/services/ai_service.go:178`）、asset 操作结果（`internal/services/asset/service.go:318-348`）、workflow source library（`internal/services/workflow/source_library.go:133-214`）、recording completed event（`internal/services/recording/service.go:365,821`）、window capture hotkey event（`internal/services/tools/window_capture_hotkey_windows.go:188-195`）。这些不是全部 Wails return error，因此不会自动经过 `apperr.Marshal`。

所有异步 event 和 result 内嵌错误也应使用同一 envelope 子集，禁止 `error: string`。否则统一 Marshal 只能覆盖同步 RPC，无法保证产品整体规范。

## Timeout 语义审计

Timeout 不是一种统一的“报错”，至少要区分三类：

1. **业务条件未满足（正常 route）**：wait-template、wait-stable/change、wait-window/gone。它们应返回 `timeout` exec output，Run 可以 succeeded；需要 status/evidence，而不是 terminal error。
2. **外部调用 deadline（失败）**：AI generation、network/adapter call 等，应返回稳定 error code + timeout 参数 + retryable 语义。
3. **Run cancel/deadline**：scheduler 将 `context.Canceled/DeadlineExceeded` 作为 attempt cancel 处理（`internal/workflow/compiler/scheduler.go:294-297`），不能误翻译成节点业务 timeout。

当前各业务 timeout 信息不一致：

- Template 会发 `automation.template.timeout` status（`internal/nodes/automation_template.go:24,117`；`internal/noderuntime/automation_template.go:187-203`），并输出匹配结果；但现有 timeline summary 还不足以完整解释最佳候选。
- Observation 的轮询返回 `changed/stable/timeout`，timeout 时仍输出最后一次 `changed-ratio/mean-difference`（`internal/noderuntime/automation_observation.go:99-128,181-193`），但 adapter summary counters 仅显式加入 captures/capture_bytes（`61-82`），也没有对应 timeout status，所以 Timeline 很难直接呈现最后观测值。
- Wait Window 在 `Matched=false` 时只返回 `ExecOutputs:["timeout"]`（`internal/noderuntime/automation_window.go:96-108`）；没有 status、最后候选、selector 摘要或耗时 evidence。
- 前端判断“未连接 timeout route”目前硬编码只认识 `automation.template.timeout`（`frontend/src/app/editor/runTrace.ts:105-107`），因此 observation/window timeout 即使存在也无法得到同等提示。

建议所有声明 timeout output 的节点遵守统一 Outcome Evidence 最小集：`outcome=timeout`、`timeout_ms`、`elapsed_ms`、`attempts/observations`，再由节点 contract 声明领域参数（template 的 best score/threshold/candidate；observation 的 last ratio/threshold；window 的 selector reason/candidate count）。应由 contract metadata 声明 status 与 route 的映射，前端不得继续硬编码某一个 status code。

## 推荐统一模型

建议只建立一个跨边界词汇，具体载体按场景裁剪：

```text
Error ID       stable machine code, e.g. automation.target_not_found
Category       validation/domain/adapter/policy/infrastructure
Params         schema-bound, locale-free, redacted values
Location       workflow/run/graphPath/node/attempt/operation
Retryability   retryable + optional retry reason
Operation ID   correlates UI/AI with logs
Cause          process-local only; never product message by default
Evidence       bounded observed facts for timeout/failure diagnosis
```

Node Contract 当前已经声明 error code/category/retry hint，并由 scheduler 强制校验；应扩展参数 schema/evidence schema，而不是让 adapter 自由返回 map。Run journal 的 `Summary` 已有 counters/facts，可作为过渡，但 failure 参数必须与普通性能 counters 分开，以免语义含混。

## 实施顺序与机械门禁

1. **定义 contract**：为 RPC、async event/result、NodeFailure、Run terminal、MCP 写同一概念模型和 JSON schema；规定 code 命名、参数类型/预算、脱敏、兼容策略。
2. **封闭泄漏**：`apperr.From` 的 unclassified message 不再面向用户返回 raw cause；Automation health 和异步 event/result 移除 `err.Error()` transport。
3. **贯通 Run params/evidence**：NodeFailure/timeout outcome 到 journal、service、export、MCP；保留 operationId 日志关联。
4. **逐节点清单审查**：从当前 Node Catalog 生成每个 declared error、status、timeout route 的表，要求 error i18n、参数 schema、行动建议、timeout evidence；CI 检查投影完整性。现有 i18n node-error parity（`frontend/src/i18n/check.cjs:248-286`）可扩展，而不是人工维护平行清单。
5. **统一 renderer**：Timeline、toast、settings health、AI diagnosis 使用同一个 code renderer；默认用户层显示行动文案，技术详情显示 code/location/operationId/params。
6. **MCP Run tools**：只消费结构化 Run Evidence；修改工作流仍走既有 typed patch/CAS，诊断建议引用 error ID 和参数。

## 验收标准

- 任一用户可见失败都能导出稳定 ID；切换中英文不依赖 Go/adapter 英文文本。
- 用户主界面不出现 package 名、adapter implementation 名、Win32/API 原始错误或路径；技术详情可凭 operationId 查日志。
- 每个 timeout output 都有结构化 evidence，并明确它是正常 route 还是 terminal failure。
- 历史 Run 在无进程日志时仍足以回答“哪个节点、什么条件、期望值、观测值、建议动作”。
- 前端和 MCP 不使用 `startsWith/includes` 解析业务错误字符串；未知 code 有安全 fallback，仍展示可复制的 code/operationId。
