# Framework extension when behavior varies by dispatch context

SUMMARY: 节点行为随 dispatch context 变时怎么扩 —— 优先在 dispatch 入口 wrap 服务，别加宽 Ctx 接口
READ WHEN: 设计 framework / DI 容器扩展 / 节点 Ctx 加新方法前 / "这种节点该看到不同的 X" 类需求
RECHECK WHEN: 改 Ctx 接口 / ServiceBundle 组成 / DI 容器装配 / 给节点 Ctx 加新方法 / 改 dispatch 入口适配层时

---

## 决策原则

**新需求**: "某类节点 (PureData / Cron / Async / 等) 看到的 X 服务行为应该跟普通节点不同" (X = Vars / Sys / time / Stop 信号 / etc.)

**两个 path**:

### ✅ Path A (推荐): adapter-layer wrap at dispatch entry

dispatch 入口侧 (engine.go 内 `RunNode` / `EvaluatePureData` / `RunNodeAsRegion` 类 entry) 把 `services.X` 用 wrapper 替换, 然后构造 Ctx. Ctx interface 不变. 节点作者写同样的 `ctx.X().Method(...)`, framework 透明保证 context-correct 行为.

```go
// engine.go::EvaluatePureData
if services.Snapshot != nil {
    snap := services.Snapshot()
    services.Vars = newSnapshotVarStore(services.Vars, snap)  // wrap, don't widen
    if snap.Sys != nil {
        services.Sys = snap.Sys
    }
}
// build Ctx with wrapped services
```

### ❌ Path B (避免): Ctx interface widening

Ctx 加新方法 (`Ctx.SnapshotVars()` / `Ctx.LiveVars()` / `Ctx.PausableSleep()`). 节点作者必须选对方法. 易写错 (Runnable 误用 SnapshotVars / Evaluator 误用 Vars). 接口一开始往这方向扩, 后续每个新 context 类型都加 2 方法.

## 为什么 A 赢

| | Path A (wrap) | Path B (widen) |
|---|---|---|
| Ctx 接口稳定 | ✓ 0 改 | ✗ 持续膨胀 |
| 节点作者负担 | ✓ 无选择责任 | ✗ 必须选对 |
| 误用风险 | ✓ 编译期不可能 | ✗ runtime 才暴露 |
| 测试 stub 复杂度 | ✓ stub framework 注入路径即可 | ✗ stub 多方法 |
| 新增 context 类型 | ✓ 加 wrapper, 0 接口变 | ✗ 接口 +N 方法 |

唯一例外: 行为差异需要节点 **显式选择** 时 (e.g. "我这个节点要 explicit live read") — 那是节点级 opt-in 语义, 应当显式. 但这种 case 少, 默认 wrap 路径.

## 怎么 apply

1. 新需求 "这类节点应看到 X 的不同行为" — **先问**: 能在 dispatch entry wrap `services.X` 吗?
2. 几乎总是能. read-only 行为约束 (snapshot / frozen / pause / blacklist) 都直接走 wrap.
3. 不能 wrap 的情况:
   - 行为差异跟节点自身参数耦合 (e.g. GetVar 的 scope 字段) — 这是节点的事, 不是 dispatch 的事. wrap 仍可, 但接受 wrapper 透传 scope.
   - 节点要主动调用一个 dispatch-only 操作 (e.g. Loop body 调用 callback) — 这是回调, 不是 service. 走另一种 pattern (RegionRunner.body).

4. wrap 实现要点:
   - **wrapper 是 framework-internal 类型**, 不暴露给节点 (lowercase 类型名, package-internal constructor).
   - **不安全写操作 panic**: e.g. snapshotVarStore.Set panic. 行为约束 invariant 通过 panic 强制, 不靠节点作者 "记得不要写".
   - **wrapper 跨包边界用 interface 承载**: e.g. `Snapshot.Sys SysStore` — 不是 raw 数据 + helper, 让 runtime 端塞已 wrap 的实例, node 端透明用.

## Case 1 — 2026-05-26 P2.2 C5

需求: PureData Evaluator 看到的 Vars/Sys 应该是 tick-frozen snapshot, 不是 live state.

初版方案 (α): Ctx 加 `SnapshotVars()` / `SnapshotSys()` 方法. brainstorm 时被否 — Ctx 接口扩张 + 节点作者必须选对.

确定方案 (α'): EvaluatePureData 入口 wrap `services.Vars` 成 `snapshotVarStore` (`internal/node/snapshot.go`), 把 `services.Sys` 替成 runtime 提供的 `frozenSysStoreAdapter`. Ctx interface 0 改. GetVar.Evaluate 写 `ctx.Vars().GetScoped(name, scope)` — 自动拿 snapshot. Runnable 同样写法拿 live. Framework 在 dispatch 路径自动分流.

Commit: `cfa82a6` (C5a 加 wrap infra) + `22b0720` (C5b cutover).
