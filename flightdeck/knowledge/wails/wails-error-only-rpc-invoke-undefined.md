# ⚠ wails error-only RPC → FE invoke() 返 undefined 不可辨成功失败

SUMMARY: error-only wails RPC 成功和失败都让 invoke() 返 undefined, 无法分支; 改返 (bool, error)
READ WHEN: 加一个"开窗 / 跑前置检查"类 wails RPC 且 FE 之后要 awaitWailsEvent 拿结果; 或撞 "RPC 失败了但前端没察觉, await 一直不返回"

---

## 症状
`OpenCalibratorHUD` 起初签名 `func(reqID string) error`。FE 流程:
```
const r = await backend.tools.openCalibratorHUD(id)  // 想据此判断是否开窗成功
const res = await awaitWailsEvent('calibration:result', p => p.id===id)  // 再等结果
```
检测不到游戏窗口时后端 `return error` —— `invoke()` 包装(`lib/invoke.ts`)catch 了 error、toast、**返 undefined**。但**成功**时 Go 方法返 `nil` error → wails 返 void → `invoke()` 也返 **undefined**。两者无法区分 → 失败时代码照样往下 `awaitWailsEvent`, 那个事件永远不来 → **promise 永久挂起 (泄漏)**。

## 根因
`invoke<R>()` 签名是 `Promise<R | undefined>`: 成功返 `R`, 失败 catch 后返 `undefined`。当 Go 方法只返 `error` 时 `R = void`, 成功也是 `undefined` —— 信息被抹平。

## 修法
带"前置条件可能失败 + 调用方要据此分支"的 RPC, Go 返 `(bool, error)` (成功 `true,nil`):
```go
func (s *Service) OpenCalibratorHUD(reqID string) (bool, error) { ... return true, nil }
```
FE: `const ok = await ...; if (!ok) return` —— 成功 `true` / 失败 `undefined` 都被 `!ok` 正确分流, 失败不会去 await 那个不会来的事件。

## 教训
- error-only RPC 只适合 "fire-and-forget / 失败有 toast 就够" 的调用 (e.g. openScreenPicker —— picker 必开, 没前置失败分支, 所以没踩到)。
- 一旦调用方**要根据成功与否决定下一步** (尤其后面挂 awaitWailsEvent), 必须让成功可辨 → 返 bool。
- 反过来 review: 看到 `await backend.x.openY(); await awaitWailsEvent(...)` 而 openY 只返 error, 警惕失败路径挂起。
