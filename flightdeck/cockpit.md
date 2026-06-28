# Cockpit — YHFish

Focus: **window-control 代码完成 + 逐任务 review + opus 整支终审过(终审抓的 Critical 已修复核)**,与 detect-click 一起**待用户真机 smoke**;两者 smoke 过即归档,然后回 ③ MCP 对外暴露(mcp-node-exec,未起)。
> point-px-unit ✅ 已发并真机 smoke 过(2026-06-25)→ 已归档。

## In flight

- [work/window-control/](work/window-control/) — **代码全完成,待真机 smoke**。给框架加一等 `Window` 数据类型 + GetWindow 产出节点 + WindowState/MoveResizeWindow/CloseWindow 控制节点 + 所有 NeedsWindow 节点可选 Window 输入(不连=当前活动窗口,连了=派发期 per-node 覆盖栈) + "Window" palette 类别 + 中英 i18n。17 任务全 subagent-driven 实现 + 逐任务 review + **opus 整支终审**。全绿门过(go build/vet/test 全 scope 绿除预存 RED 基线;前端 vue-tsc + i18n parity)。**终审抓 1 Critical(Snapshot 返缓存未按 design live 重读 ClientW/H,被节点 stub 假造掩盖的跨任务缝)+ 1 Important(Window pin 泄进 Script JS 全局)→ 已修(commit 28a86ca)+ re-review 干净**。进度账本 + 终审记录 `.superpowers/sdd/progress.md`;教训落 [knowledge/nodes/contract-verification-traps.md](knowledge/nodes/contract-verification-traps.md)。**待用户真机 smoke(清单见 ## Next),全过 → 移 work/window-control 到 cold store + design/plan 归档。**
- [work/detect-click-config/](work/detect-click-config/) — **代码完成,待真机 smoke**(本会话未动)。Phase 1-4 全家桶(Vision/ClickTemplate/新节点群/WaitWindowGone/Point 手填)。两已修 bug 待重测:⑦ InputText 改 WM_CHAR targeted;短图日志 LogMerger finalize 时 emit。全过 → 移 cold store。
- [work/mcp-node-exec/](work/mcp-node-exec/) — ③ MCP 对外暴露(AI 调我们):GUI 内置 Streamable HTTP MCP server,design.md + plan.md 就绪。detect-click + window-control 完结后的下一主攻。

## Next

1. **window-control 真机 smoke(用户)** —— 先 `task build` 出 exe,再验(plan D3 Step4):① 侧栏/右键/explorer 三处能找到 Window 组的 GetWindow/WindowState/MoveResizeWindow/CloseWindow;② 单窗口图(WindowTarget→WindowState 最大化)不连 Window 输入照常作用当前窗口;③ 多窗口图(GetWindow 主/子各绑变量 → 两个 WindowState 各连 GetVar 的 Window 输出)分别作用对应窗口;④ 无边框全屏→MoveResize→退无边框 回到全屏前布局;⑤ CloseWindow 接 WaitWindowGone 确认关闭;⑥ sendinput 后端下 ClickAt 连子窗口 Window 输入能打到子窗口(覆盖期补前台生效)。
2. **detect-click 真机 smoke(用户)** —— Phase 1-4 各项(尤其重测 ⑦ 记事本/vscode 打字、短图日志、WaitWindowGone、Swipe 手填/连点)。
3. 两支 smoke 全过 → 归档 work/(移 cold store + design/plan 按日期归档),回 ③ 从 [work/mcp-node-exec/](work/mcp-node-exec/) 恢复,再按其 index 指向的 plan 起 MCP 实现。

## Open questions

- **破坏性大升级决策基线(2026-06-29)**: 先不全量换 Rust,Go 保持主运行时,Rust 只下沉 Win32/native controller hot path;先引入 `Target/Controller/CoordinateSpace/Trace` 再谈 Android 和输入后端矩阵。target/controller topic 已推进到 Phase 51: Android ADB 目标节点、ADB discovery、target-aware vision、前端 async-dropdown、BrowserTarget、Browser CDP discovery/client provider、async option metadata apply、stale CDP client invalidation、runtime 快测、前端 Vitest 隔离、async dropdown 契约、active target 不回退旧 HWND 契约、节点/参数/dropdown option i18n guard、全量节点注册入口 `internal/nodes/all`、app/runtime 全量路径共用中心注册、节点注册漂移 guard、build 知识基线刷新、前端静态 i18n key 引用 guard、前端连边缺 handle 拒绝持久化、未知 kind 不渲染 fake pin、exec-out PascalCase spec guard、String default 类型 guard、Bool default 类型 guard、Point/Rect default 类型 guard、inline 连接缺 handle fail-closed 都已落地;下一刀回到全局验证与剩余迁移质量硬化。topic 入口见 [work/target-controller-upgrade/](work/target-controller-upgrade/);调研与路线见 [knowledge/architecture/automation-framework-survey.md](knowledge/architecture/automation-framework-survey.md) + [knowledge/architecture/target-controller-upgrade-guide.md](knowledge/architecture/target-controller-upgrade-guide.md)。
- **window-control 终审 Minor 池**(opus 判多数 acceptable debt,留收尾酌情):B1 gwlStyleIdx 机制注释、A1 测试无 break、C3 `r.rt.Container!=nil` 冗余守卫、C4 bring_foreground NeedsForeground 冗余无害、D2 GetWindow en `Fail` vs `Failed`、A2 GetWindow 无 NotFound 测试 等;详见 `.superpowers/sdd/progress.md` Minor 池。皆非阻塞。
- **detect-click 终审两个已知局限**(已落 knowledge,等 demand):① Swipe 在 sendinput 后端走 PostMessage、读 RawInput 的游戏收不到拖拽;② pkg/input SendInput 原语不查注入数、失败上报不到节点层。
- **当前验证基线**(2026-06-29 已刷新):`go test ./...`、`cd frontend && pnpm i18n:check`、`pnpm -C frontend test`、`cd frontend && pnpm test` 都应绿。旧 runtime fixture / i18n residue 预存红记录已过期;新红先按回归处理。见 [knowledge/build/build.md](knowledge/build/build.md)。
- **MCP/AI(③)挂账**:A6/C7 polish 余项 + AI 配置待 smoke 项,细节在 cold archive `2026-06-23-local-ai-config`。AI 系统知识 → [knowledge/nodes/ai-nodes.md](knowledge/nodes/ai-nodes.md) + [knowledge/nodes/held-exec-outputs.md](knowledge/nodes/held-exec-outputs.md)。
- **积压路由**:编辑器 footgun / i18n residue 清理等零散项在 cold store `ideas/`。
