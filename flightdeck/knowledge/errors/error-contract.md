# Error contract development

Yotta 的用户可见失败必须沿一条完整链路投影：domain cause → stable Problem → Wails envelope → frontend
`RPCError` → 当前任务的反馈表面。任一层回退成原始字符串、`undefined` 或无 ID 的异常，都是错误契约缺陷，
不能把“发生未知错误”视为可接受的产品结果。

## Canonical model

跨 Wails seam 的唯一表示由 `internal/apperr.Envelope` 定义：

- `id`：稳定、locale-free 的机器标识；翻译和调用方分支只依赖它。
- `category`：`validation | domain | policy | adapter | infrastructure`。
- `params`：有界、类型稳定、可安全展示的参数；不得包含凭据、原始 provider body、私有路径或 Go error。
- `operationId`：一次失败的关联标识；`apperr` 在缺失时生成。
- `runId`：失败属于一次 Run 时携带；不能用日志文本反查 Run。
- `retryable`：机器可读重试语义，不从中文或英文 message 推断。

`internal/apperr.Error` 适合直接声明稳定 Problem。已有 domain error 应实现
`apperr.EnvelopeProvider`，或在所属 application service 中通过 `errors.Is/As` 映射后再跨 seam。原始 cause
只留在进程内诊断上下文；`fmt.Errorf("%w: %v", problem, cause)` 可以同时保留 Problem 和 cause，但后者不得
进入 envelope params。

Workflow 节点的 declared error/status 属于 Node Contract 和 Run Evidence；应用命令失败属于 Problem。两者
可以共享稳定语义，但不能把节点 error route 当作 RPC error，也不能把 RPC message 写进 Timeline 冒充节点证据。

## Projection ownership

错误在最了解语义的 seam 映射一次：

1. Domain/adapter 返回 typed error，保留 `errors.Is/As` 能力。
2. `internal/services/` 覆盖该命令的全部阶段：参数解析、profile/target 解析、provider 创建、执行、持久化和
   收尾。不能只映射“主调用”之后的错误，而遗漏前置失败。
3. `internal/desktopapp` 只安装 `apperr.Marshal` 和安全 observer，不维护按 domain 分支的第二套映射表。
4. `frontend/src/lib/invoke.ts` 只负责兼容 native/dev transport、normalize envelope 并抛出 `RPCError`；不 toast，
   不吞错，不把失败变成 `false`/`undefined`。
5. 页面或 controller 根据任务选择 inline/page/toast/modal，并通过 `error.<id>` 翻译 `params`。

公开 service 方法返回原始 `error` 前必须逐个审查所有 early return。新增一个 adapter/provider/profile 分支时，
同时新增对应 Problem ID；“稍后主路径会统一处理”不覆盖在它之前返回的错误。

## Durable and asynchronous failures

长任务、AI 会话、Run 和后台操作不能只依靠 rejected Promise。命令开始后应尽早建立 durable identity，并让
失败证据关联到该 identity：

- Run：Journal/Timeline 保存 node outcome、Problem params 和 source revision。
- AI conversation：先保存 user turn；随后保存 assistant result，或保存 `problemId + operationId`。模型解析、
  provider 启动等前置失败也必须留下 turn，不能产生空会话和丢失原因。
- Event-driven command：RPC reject 后立即停止等待 event；进度 event 使用稳定 stage/kind，最终失败仍以
  rejected typed Problem 为权威。
- 日志：记录 `problemId`、`operationId`、`runId` 和内部 error type/cause；日志补充维护上下文，不替代用户
  反馈、Timeline 或会话证据。

持久记录保存 `problemId/params/operationId`，展示时按当前 locale 翻译；不要持久化中文/英文错误句子作为
机器事实。

## Frontend failure policy

`errorMessage()` 收到合法 envelope 时必须显示 `error.<id>`；未知 ID 显示 ID，而不是丢成通用文案。

`normalizeError()` 无法得到 `id` 表示 transport 或调用代码违反契约。调用表面必须提供阶段化 fallback，例如
“加载对话失败”“发送请求未返回结构化结果”，并保留 `RPCError.operation/operationId/source` 给诊断路径。
`error.UNKNOWN_ERROR` 只用于暴露尚未治理的缺口，不是完成态；任何真实验收出现它，都必须补测试和映射后
才能交付。

错误反馈必须说明三件事：失败对象/阶段、用户可采取的恢复动作、可用于关联的 ID。恢复动作必须对应当前
产品中真实存在且可找到的入口；没有“诊断信息”页面或跳转按钮时，不得让用户“打开诊断信息”。不要向用户显示
`adapter xxx`、Go type、provider raw body、堆栈或本机路径。

## ID and params rules

- 新 ID 使用稳定 domain namespace，例如 `ai.authoring.provider_unavailable`；不要把函数名、包名或当前实现
  写进 ID。
- 同一用户语义复用一个 ID；只有恢复动作、retryable 或安全参数结构不同才拆分。
- params key 固定且可翻译，值限制为 string/bool/有限数值或经过约束的短列表。
- timeout、not-found、conflict、cancelled、capacity、unavailable 分开；不能全部折叠为 `failed`。
- 删除或改名错误 ID 属于 compatibility change；按 `docs/compatibility.md` 审查。

## Required verification

修改错误路径时至少覆盖以下相邻门禁：

1. Domain/service table test：每个 early return 都断言 exact `id/category/params/retryable`，并断言 raw cause、
   secret 和路径没有进入 `apperr.Marshal` 输出。
2. Transport test：native object envelope 与 dev fetch JSON-string envelope 都能被 `normalizeError()` 解码；无
   envelope 异常触发阶段化 fallback。
3. UI test：断言用户可见的恢复文案、操作 ID 和输入保留，不只断言 catch 被调用。
4. Durable test：失败重启后仍能从 Run/Conversation 读到同一 Problem；前置失败不会留下无法解释的空记录。
5. `task check`：验证 bindings、i18n parity、错误 key 和相关 Go/frontend tests。
6. 涉及 Wails、event、WebView 或真实 adapter 时运行 `task webview:smoke`；provider/设备能力再运行受控的
   native smoke。mock 成功不能证明进程启动、PATH、登录或实际 transport。

Review 时从用户动作反向走完整路径：点击后可能在哪一行最早失败？每条分支是否都有 stable Problem、持久
证据、翻译和恢复动作？只检查最终 provider/adapter 的 error mapping 不算完成。

## Current implementation anchors

- Canonical envelope 与 safe observer：`internal/apperr/apperr.go`
- Wails composition：`internal/desktopapp/desktop.go`
- Frontend normalize/RPCError：`frontend/src/lib/invoke.ts`
- Typed frontend facade：`frontend/src/lib/backend.ts`
- Node declared error/status：`internal/nodecontract/`、`internal/noderuntime/`
- Run evidence：`internal/run/`
- AI conversation evidence：`internal/aiauthoring/`
- 翻译：`frontend/src/i18n/zh.ts`、`frontend/src/i18n/en.ts`
