# Wails void RPC 必须用 Promise 状态区分成功和失败

## Contract

- Go `func Command(...) error` 在成功时映射为 resolved `Promise<void>`，值为 `undefined`。
- 失败必须映射为 rejected `Promise`，原因是 typed `RPCError`；不得捕获后返回 `undefined` 或 `false`。
- `bool` 只表达真实业务结果，不能用来补偿吞错 transport。
- 调用方需要继续等待 event 时，只有 command resolve 后才能注册/等待；command reject 必须停止流程并由该 domain action 决定 inline、modal 或 failure toast。

## Stable seam

Go 在 Wails composition root 的 `application.Options.MarshalError` 安装唯一 marshaler，把所有错误投影为：

- `code`
- `category`
- locale-free `message`
- safe `details`
- `operationId` 和可选 `runId`
- `retryable`

前端只有两个模块允许直接 import generated service bindings：通用 `lib/backend.ts` 和 workflow transport。两者都通过 `invoke`/`callRPC` 解码并 rethrow `RPCError`，没有 toast side effect，也没有 failure sentinel。

## Failure pattern

```ts
try {
  await backend.tools.openCalibratorHUD(id)
  const result = await awaitResult(id)
  applyResult(result)
} catch (error) {
  showDomainFailure(error)
}
```

如果 wrapper 把失败变成 `undefined`，void success 与 failure 会重新合并，后续 event wait 可能永久挂起，录制 finalize 还会在原始错误之后制造 `invalid result` 等假错误。

## Verification

- transport test 断言失败 reject `RPCError` 且没有自动 toast。
- static test 禁止 `invokeVoid`、wrapped RPC 的 `Promise<T | undefined>` 和 browser-native alert。
- domain test 断言 command reject 后不会进入后续 event wait 或结果校验。
