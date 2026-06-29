# ⚠ LL keyboard hook 停录热键去抖 — atomic.Swap 一次性消费

SUMMARY: LL keyboard hook 停录热键 auto-repeat 重复 fire, 用 atomic.Swap 一次性消费去抖
READ WHEN: 写 LL keyboard / mouse hook 接全局热键; callback 里跑业务逻辑而不只是消费事件; 停录 / 全局快捷键看到"第一次成功但 toast 是 error"

---

## 教训 (3 条)

1. **Windows LL keyboard hook 一次 keydown 触发一次 callback, 但按住会 auto-repeat (~30Hz)**. 如果 callback 启 goroutine 跑业务 (`go (*cb)()`), N 个 goroutine 同时排队 — 第一个赢的跑完, 后面的撞已停状态.

2. **去抖在 hook 层做, 不在业务层**. `activeStopCallback.Swap(nil)` 原子 load-and-store, 第一次拿到 cb 非 nil → fire; 后续都拿到 nil → 静默 drop. 一次 Start session 内, callback 最多 fire 一次.

3. **业务层加哨兵 silently swallow 是 defense-in-depth**. `Service.StopAsync` 收到 `ErrRecorderNotActive` 时不 emit error 事件 — 即使有别的路径 (toolbar / HUD 按钮 + F12 同时) 撞已停状态, 也不会跳 toast 覆盖前面 success 的事件.

## 撞了什么

用户报 "录制结束提示 recorder not active". 走完一次成功停录后, 前端 toast 显示的是 error 而不是 success.

排查链:
- `Service.Stop()` 在 `s.rec.Active()` false 时 return `"recorder not active"` 错.
- `StopAsync()` 把错 emit 成 `recording:completed {error: "..."}` 事件给前端.
- 前端 `useRecording` 的 listener 看到 error 字段 → 跳 error toast.

根因: F12 keydown auto-repeat → `keyboardProc` 反复检测停录热键 → 反复 `go (*cb)()` spawn StopAsync goroutine → 第一条 acquire `s.mu` → rec.Stop 成功 → emit success → 第二条 acquire mu → `rec.Active()=false` → emit error.

## 怎么修

`internal/services/recording/llhook.go:241`:

```go
// 之前: Load 后留指针, 后续 keydown 还能再 fire
if cbp := activeStopCallback.Load(); cbp != nil {
    go (*cbp)()
}

// 现在: Swap 一次性消费, 后续 fire nil
if cbp := activeStopCallback.Swap(nil); cbp != nil {
    go (*cbp)()
}
```

外加 `internal/services/recording/recorder.go` 导出 `ErrRecorderNotActive` 哨兵, `service.go:StopAsync` 静默吞:

```go
if err != nil {
    if errors.Is(err, ErrRecorderNotActive) {
        return // 已被另一条路径停了, 别覆盖前面 success emit
    }
    s.emit("recording:completed", map[string]any{"error": err.Error()})
    return
}
```

`Service.Stop()` (sync RPC 路径, toolbar 调) 保持原行为返错 — toolbar 同步 caller 需要明确信号, 前端 `stopRecording()` 入门也已经 check `recordStore.isRecording` 短路, 不需要再吞.

## 为什么 Swap 而不是 Load

- Load 是非破坏性 — 反复 Load 反复拿到同一指针.
- Swap(nil) 是 atomic load-then-store-nil, 单 hook 线程 (LL hook callback 在固定 OS 线程) 也保证原子性: 即使 hook callback 真并发 (多 hook 实例) 也只一次 hit.
- 同 session 想"恢复" callback 需要 Service.Start 再 SetActiveStopHotkey() — 正常生命周期就是这样, 不破坏.

## 为什么 callback 不在 hook 线程直接跑

- LL hook callback 必须 fast return (~300ms 超时 OS 自动 unhook). Stop 走 PostQuitToThread + 等 worker 退 + drain channel, 远超时.
- 所以 callback 必须 spawn goroutine 异步跑 — 这才是 race 的入口.

## 怎么防

- 全局热键 callback 默认走"一次性消费"模式, 除非业务明确需要多次触发.
- LL hook 处理停录类不可逆动作时, 别只在业务层判幂等 — hook 层先吞掉重复 fire 才省 goroutine + s.mu 抢占.
- 业务层加哨兵错 + 显式静默路径, 不要靠 toast 跳出来.
