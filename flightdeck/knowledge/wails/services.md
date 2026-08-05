# Wails service and transport development

## Backend seam

`internal/services/` 是 application service/DTO adapter，不拥有第二套 Workflow runtime，也不 import Wails。
Wails-specific composition、window 和 event wiring 位于 `internal/desktopapp/desktop.go`；storage-backed 核心由
`internal/localruntime` 提供。

新增或修改 service：

1. 在 `internal/services/<domain>/` 实现窄 service 和 tests。持久化调用所属 Repository/Store，不从 service
   派生 profile 路径、不暴露 `*sql.DB`、不自行创建 compiler/executor。
2. 导出方法只使用 bindings 可表达的 DTO；不把 secret、interface、function、native handle 或 ambient
   authority 传到前端。
3. 在 `internal/desktopapp/desktop.go` 的 composition 中构造、注册，并把 lifecycle-owning 组件交给
   `appruntime` 关闭。新增 registration 后必须重启后端，前端 HMR 不会新增 RPC。
4. 通过正式 Task/Wails 入口重新生成 `frontend/bindings/`；它被 gitignore，不能手改或提交。
5. 普通 service 在 `frontend/src/lib/backend.ts` 增加 typed facade；Workflow command 在
   `frontend/src/app/transport/workflow.ts`。其它 store/component 不直接 import generated bindings。

## Error and event contract

- Go `func Command(...) error` 成功对应 resolved `Promise<void>`，失败必须 rejected typed RPC error。wrapper
  不得 catch 后返回 `undefined`、`false` 或 toast；否则 void success 与 failure 无法区分。
- `frontend/src/lib/invoke.ts` 统一处理 native 与 dev fetch transport 的 error envelope 并 rethrow；domain
  action 按[UI 指南](../frontend/ui.md)选择 inline/page/toast/modal。
- command resolve 之后才能等待后续 event；command reject 必须停止 event wait，避免永久等待或制造二次
  “invalid result”错误。
- event payload 使用可校验 DTO。带 generation/sequence 的 snapshot 只允许单调前进；较晚返回的 RPC
  不能覆盖已经收到的更新 event。
- service/event 名与签名由 tracked RPC contract 审查；某次 service/method 数量只是观测值，不是门禁。

## Verification

- Go service tests 覆盖 validation、authority、持久化/Repository 和 shutdown/error path。
- frontend transport tests 断言 success resolve、failure reject、event decode 和 stale snapshot rejection。
- 运行 `task check` 让 bindings contract 与前后端门禁按路径路由；涉及窗口、event 或实际 WebView transport
  再运行 `task webview:smoke` 并重启真正后端。
