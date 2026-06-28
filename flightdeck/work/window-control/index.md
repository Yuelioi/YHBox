# Index — window-control

## State

代码实现、逐任务 review、opus 整支终审都已完成。终审抓到的 Critical/Important 已修复并 re-review 干净；当前只等用户真机 smoke。smoke 全过后归档本 topic。

## Next

用户真机 smoke：
- `task build` 出 exe。
- 验证 Window 组入口：侧栏、右键、explorer 三处能找到 GetWindow / WindowState / MoveResizeWindow / CloseWindow。
- 单窗口图：WindowTarget -> WindowState 最大化，不连 Window 输入时仍作用当前活动窗口。
- 多窗口图：GetWindow 主/子各绑变量，两个 WindowState 各连 GetVar 的 Window 输出，分别作用对应窗口。
- 无边框全屏 -> MoveResize -> 退无边框，回到全屏前布局。
- CloseWindow 接 WaitWindowGone，确认关闭。
- sendinput 后端下 ClickAt 连子窗口 Window 输入，能打到子窗口。

## Read now

- knowledge/build/build.md — build / smoke 前置约定与预存失败基线。

## Read if

- design.md — 如果 smoke 暴露设计语义冲突，或要判断 Window 覆盖栈 / 控制节点契约。
- plan.md — 如果要追实现任务、验证命令或历史任务边界。
- knowledge/nodes/contract-verification-traps.md — 如果测试全绿但真实 adapter 行为不符 design。

## Progress

Done:
- Window 一等数据类型、GetWindow、WindowTarget 产出 Window。
- WindowState / MoveResizeWindow / CloseWindow。
- NeedsWindow 节点可选 Window 输入与派发期 per-node 覆盖。
- Window palette 分组与中英 i18n。
- 17 个实现任务、逐任务 review、整支终审。
- Critical Snapshot live 重读和 Important Script Window pin 泄漏已修。

Verified:
- go build / vet / test scope 通过，除 cockpit 记录的预存 RED 基线。
- frontend vue-tsc 与 i18n parity 通过。

Remaining:
- 用户真机 smoke。

## Open questions

- 终审 Minor 池为可接受 debt，非 smoke 阻塞：gwlStyleIdx 注释、测试 break、冗余守卫、GetWindow 文案和 NotFound 测试等。
