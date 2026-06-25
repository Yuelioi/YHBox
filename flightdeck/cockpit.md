# Cockpit — YHFish

Focus: detect/click 能力扩展（Phase 1-4）代码完成、终审过，**待用户真机 smoke**；smoke 后回 ③ MCP 对外暴露（mcp-node-exec，未起）。
> **point-px-unit ✅ 已发并真机 smoke 过（2026-06-25）** → 归档 `archive/specs|plans/2026-06-25-point-px-unit*` + 知识 [knowledge/frontend/screen-pick-coverage.md](knowledge/frontend/screen-pick-coverage.md)。

## In flight

- [work/detect-click-config/](work/detect-click-config/) — **代码完成，待真机 smoke**。Phase 1 Vision 基础层 · Phase 2 ClickTemplate 全家桶 · Phase 3 新节点群（WaitTemplateGone · Swipe · Scroll 横向 · InputText · StopApp · ClickAt 组合键+多击）· **Phase 4（用户增补）**：WaitWindowGone（等窗口关闭，对称 WaitWindow，配 StopApp 确认真关掉）+ Point 类型手填控件（PointWidget x/y 百分比；Swipe 起止点在 Inspector 手填或连检测节点点输出）。逐 task TDD + 每 task review + 整支 opus 终审，build/vet/全 scope 全绿（除已知 runtime RED 基线）。plan-phase1..4 + 进度账本 .superpowers/sdd/progress.md。**真机 smoke 进行中**，已修两个 bug（均 build/vet/test 全绿，**待用户真机重测**）：(1) ⑦ InputText 在 postmessage 后端旧实现走全局 SendInput → 后台目标窗口收不到，改走 PostMessage WM_CHAR targeted → [knowledge/input/postmessage-typetext-uses-wm-char.md](knowledge/input/postmessage-typetext-uses-wm-char.md)；(2) 节点勾 logEnabled 短图经常没日志 —— LogMerger.finalizeLocked 只写文件不 emit，短图在 250ms tick 前跑完则前端面板零日志（文件里有），改 finalize 时 emit final → [knowledge/logging/short-run-flush-loses-dump.md](knowledge/logging/short-run-flush-loses-dump.md)。其余 smoke 项仍待用户验，全过即移 cold store。
- [work/mcp-node-exec/](work/mcp-node-exec/) — ③ MCP 对外暴露（AI 调我们）：GUI 内置 Streamable HTTP MCP server，暴露 run_node/find_window/写图四件套，design.md + plan.md 就绪。detect-click 完结后的下一主攻（原 focus，本会话未动）。
- [work/window-control/](work/window-control/) — **窗口能力扩展，design.md 已落 + 三家 AI 评审纳入 + 用户裁定，待用户终审 → 转 writing-plans**。定案：① 加 `Window` 一等类型（仿 Image，只读 preview，运行期瞬时不序列化）；② 两个产出节点 `GetWindow`（解析→Window，**不**改活动窗口）+ `WindowTarget`（设活动窗口 + 附带产出 Window）；③ 给所有 NeedsWindow 节点经**显式共享 helper** `WindowInputSpec()` 加**可选** `Window` 输入（不连=当前活动窗口零改动，连了=派发期 **per-node 覆盖字段+defer**、不碰粘性窗口）；④ WindowState（5 态，无边框存原布局可还原）+ MoveResizeWindow + CloseWindow（发送关闭请求，接 WaitWindowGone 确认）；⑤ "Window" palette 类别。**两轮**评审 decided：覆盖改 **per-node 栈+defer** / sendinput 补前台靠新 `Spec.NeedsForeground` 标志 / hwnd 失效→WINDOW_INVALID / 连了取不到→报错不回落 / 无边框存原布局可还原+PID 防 HWND 复用 / 控制节点 Done.Window **重读元数据**（CloseWindow 不透传）/ `WindowInputSpec()` + **Register 强校验** NeedsWindow⟹有 Window 输入 / 不留兼容。三家均判「大坑已平、余为细化」，design 收敛。下一步：用户终审 → writing-plans。

## Next

1. **detect-click 真机 smoke（用户）** —— 当前活跃线。Phase 1-3：① 锚点偏移点中 · ⑤ 多命中点最上/第2 · ⑧ 限 ROI · ② 等图消失 · ③ ctrl+点 · ④ 双击 · ⑥ 拖滑块 · ⑦ 搜索框打字（已修 WM_CHAR，重测：记事本应能打字，vscode 也试）· ⑨ 横向滚 · ⑩ 杀进程；**Phase 4**：WaitWindowGone（开记事本→等其窗口关）· Swipe 在 Inspector 手填 x/y 拖拽 · Swipe 连 ClickTemplate 的点拖拽。全过 → 移 work/detect-click-config 到 cold store。
2. **window-control（并行新线）**：用户过目 [work/window-control/design.md](work/window-control/design.md) → 改无异议则转 writing-plans 出实现计划 → 落地（Window 类型 + WindowTarget 产出 + 派发期可选窗口覆盖 + 三个控制节点 + Window 类别）。
3. 之后：按 [work/mcp-node-exec/plan.md](work/mcp-node-exec/plan.md) 起 MCP 实现（winutil.EnumTopWindows → ContainerRunner.ExecOutputs 访问器 → internal/services/mcpserver 包 → settings arm 开关 → main.go HTTP server 生命周期 → 设置页 MCP tab → 退役 cmd/yotta-mcp）。

## Open questions

- **detect-click 终审两个已知局限（已落 knowledge，未修、等 demand）**：① Swipe 在 sendinput 后端实际走 PostMessage、读 RawInput 的游戏收不到拖拽 → [knowledge/input/sendinput-drag-uses-postmessage.md](knowledge/input/sendinput-drag-uses-postmessage.md)；② pkg/input 的 SendInput 原语不查注入数、失败上报不到节点层（InputService.error 在 sendinput 后端恒 nil）→ [knowledge/input/sendinput-primitives-ignore-failure.md](knowledge/input/sendinput-primitives-ignore-failure.md)。皆全包/共享原语既有模式、非 Phase 3 回归，整支终审裁定 ship-as-is + 名状在案。
- **预存失败基线**（跑测试判红按此排除）：runtime 缺 fish fixture（apply_direction.json / watchdog_check.json）→ `TestApplyDirection_*` / `TestWatchdog_*` / `TestFishingV2Main_StateCycleSmoke` 恒红；i18n residue / `pnpm lint` 18，见 [knowledge/build/build.md](knowledge/build/build.md)。
- **MCP/AI（③）挂账**：A6/C7 polish 余项（AI/Image 节点分组 · Model combobox · 删连接确认弹窗 FE · SaveImage/LoadImage 编辑期路径校验）+ AI 配置 ① 待 smoke 项（Anthropic 原生 / 本地 Ollama / 删默认清空 / 重启持久化），细节在 ① 配置 spec §9（cold store `archive/specs/2026-06-23-local-ai-config.md`）。AI 系统知识 → [knowledge/nodes/ai-nodes.md](knowledge/nodes/ai-nodes.md) + [knowledge/nodes/held-exec-outputs.md](knowledge/nodes/held-exec-outputs.md)。
- **积压路由**：编辑器 footgun / i18n residue 清理等零散项在 cold store `ideas/`（editor-footgun-backlog.md / misc-tools-backlog.md）。
