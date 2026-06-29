# Index — window-control

## State

代码实现、逐任务 review、opus 整支终审都已完成。终审抓到的 Critical/Important 已修复并 re-review 干净；用户真机 smoke 暴露 1 个前端创建节点回归：连续新增两个 `Win32WindowTarget` 时浅拷贝 defaults 导致嵌套 `config.literal` 共享，改一个目标窗口会同步到另一个。已修复 `useInlineMenu` / `useNodeCreation` 默认 config 深拷贝并加前端回归测试。当前待用户重测该项及剩余 smoke。smoke 全过后归档本 topic。

## Next

用户真机 smoke：
- `task build` 出 exe。
- 重测连续创建两个 Windows 目标窗口：第一个绑定 AE 主窗口，第二个绑定合成设置；修改/捕获其中一个不应同步改另一个。
- 验证 Window 组入口：侧栏、右键、explorer 三处能找到 GetWindow / WindowState / MoveResizeWindow / CloseWindow。
- 单窗口图：Win32WindowTarget -> WindowState 最大化，不连 Window 输入时仍作用当前活动窗口。
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
- Window 一等数据类型、GetWindow、Win32WindowTarget 产出 Window。
- WindowState / MoveResizeWindow / CloseWindow。
- NeedsWindow 节点可选 Window 输入与派发期 per-node 覆盖。
- Window palette 分组与中英 i18n。
- 17 个实现任务、逐任务 review、整支终审。
- Critical Snapshot live 重读和 Important Script Window pin 泄漏已修。
- 真机 smoke 发现并修复：新增多个 `Win32WindowTarget` 时默认 `config.literal` 共享引用，导致两个目标窗口配置同步变化。根因与守卫见 `knowledge/frontend/node-default-config-shared-reference.md`。
- 影响面审计：静态 grep 后，普通 graph node 新增入口只剩 `useInlineMenu` 与 `useNodeCreation` 读取 defaults，均已深拷贝；`onFixMissingWin32WindowTarget` 原本深拷贝。理论上旧 bug 可影响任何带默认 literal 的节点同会话连续新增，但修复后不再局限或遗留在其他普通新增入口。落盘扫描 `bin/data/containers/**/container.json` 只发现 1 个可疑业务相关重复：当前容器两个 `Win32WindowTarget` literal 一样；另一个 `fishing-v2` 两个 SetVar 都写 `state=IDLE`，看起来是合理业务重复。

Verified:
- go build / vet / test scope 通过，除 cockpit 记录的预存 RED 基线。
- frontend vue-tsc 与 i18n parity 通过。
- 2026-06-29: `pnpm -C frontend exec vitest run src/composables/containerEditor/useInlineMenu.test.ts src/composables/containerEditor/useNodeCreation.test.ts` 通过；`pnpm -C frontend test` 通过；`pnpm -C frontend typecheck` 通过。
- 2026-06-29 impact audit: `git grep` defaults 创建入口；PowerShell 扫描 `bin/data/containers/**/container.json` 同 kind 同 literal 重复。

Remaining:
- 用户真机 smoke：先重测两个 Windows 目标窗口独立绑定，再继续原 smoke 清单。

## Open questions

- 终审 Minor 池为可接受 debt，非 smoke 阻塞：gwlStyleIdx 注释、测试 break、冗余守卫、GetWindow 文案和 NotFound 测试等。
