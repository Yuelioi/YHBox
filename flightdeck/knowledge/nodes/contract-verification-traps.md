---
kind: trap
summary: "全节点不变式别靠文本 grep(漏对齐空格), 用遍历 node.All() 的语义守卫; 节点测试 stub 服务会掩盖真 adapter 没实现 design 要求的行为, 需在 runtime 包补 adapter 级测试。"
activation: symptom
read_when: "给一批/全部节点加统一字段或输入(全节点 sweep) · 写「每个 NeedsX 节点都该有 Y」类不变式 · 节点测试 stub 了某 service 而该 service 的真 adapter 有 design 强制行为(如重读/刷新) · 排查「单任务测试全绿但跨任务集成出错」时"
---
# ⚠ 节点/适配器契约校验的两个假绿陷阱
window-control(2026-06-25)两个被终审才抓到的坑,都是「局部看着绿、整体是错的」:

## 1. 全节点不变式用语义守卫, 别用文本 grep

C4 要给所有 `NeedsWindow` 节点加 Window 输入。控制端用 `grep "NeedsWindow: true"`(**单空格**)列清单 —— 漏了 `script.go` 的 `NeedsWindow:   true`(**对齐多空格**)。25 个里漏 1 个,build/vet 都不报(漏的那个只是没 Window 输入,编译合法)。

**救场的是语义守卫**:`TestAllNeedsWindowNodesHaveWindowInput` 遍历 `node.All()` 逐个查 `Spec.NeedsWindow ⟹ 有 Window 输入`,在 init() 注册全节点的包(`internal/services/container/runtime`,其 `dispatch_v5_test.go` blank-import 全部 node 包)里跑 → 立刻抓出 Script。

要点:
- 「每个满足 X 的节点都该有 Y」这类全集校验,**靠遍历 `node.All()` 的测试 + Register init-panic 不变式**(registry.go),不靠脆弱文本 grep。
- 该测试**必须放在 blank-import 了全部 node 包的测试包**(runtime,经 dispatch_v5_test.go);放 `internal/node` 包没用(它不能 import nodes/*,会循环 → All() 为空 → 假绿)。
- Register 的 init-panic 不变式是最强兜底:任何引入这些 node 包的二进制启动即炸,无处可藏。但要注意**落地时机**:不变式要等所有现存节点都满足后才能加(否则现存节点 init() 全 panic 击穿全仓)—— 故 window-control 把「helper+字段」(早)与「Register 不变式+守卫」(晚, 全节点补齐后)拆成两步。

## 2. 节点测试 stub 服务 → 真 adapter 的 design 行为没人验

`WindowState`/`MoveResizeWindow` 的 `Done.Window` 按 design §4.4 必须**操作后 live 重读** ClientW/H(maximize/move 改了尺寸,透传旧值 = 下游拿过期尺寸)。

- 节点测试用 `recordingWindowService` stub,其 `Snapshot()` 返回一个**手设的** `node.Window{ClientW:1920}` —— stub **假造**了「fresh」,所以节点层测试全绿。
- 但真 adapter `windowAdapter.Snapshot()`(在**另一个任务/另一个包** runtime)实现成返回**缓存**的 `rt.WindowHandle()`,**从不重读**。逐任务 scoped review 各看各的,没人对到这条缝。
- 整支终审(opus 跨任务)才抓出来。

要点:
- 节点测试 stub 一个 service 时,**那个 service 的真 adapter 是独立契约**,stub 通过 ≠ adapter 正确。design 强制的 adapter 行为(重读/刷新/失效处理)要在 **runtime 包对真 adapter 写测试**(注入 seam 如 `clientSizeFn`/`resolveWindowFn`/`isWindowFn` 替身,断言真 adapter 走了该路径)。
- 写节点任务的 brief/spec 时,凡 design 对某 service 方法有强制语义,**把该语义同时写进 adapter 任务的验收**,别只测节点调了该方法。
- 通用信号:**单任务全绿但整体可能错** → 终审(整支跨任务 review)是唯一能抓这种缝的网,别省。

相关: [held-exec-outputs](held-exec-outputs.md)(Window 值经 held output 直连流动) · 注入 seam 模式见 `node_services.go` 的 `resolveWindowFn`/`clientSizeFn`、`dispatch_v5.go` 的 `isWindowFn`。
