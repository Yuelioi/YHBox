# Cockpit — YHFish

Focus: detect/click 能力扩展全落地（Phase 1/2/3 = spec 10 项 + Phase 4 = 用户 review 后增补 WaitWindowGone + Point 手填控件）—— build/test 全绿、逐 task review + 整支 opus 终审过；仅剩用户真机 smoke；smoke 后回到 ③ MCP 对外暴露（mcp-node-exec，未起）。

## In flight

- [work/detect-click-config/](work/detect-click-config/) — **代码完成，待真机 smoke**。Phase 1 Vision 基础层 · Phase 2 ClickTemplate 全家桶 · Phase 3 新节点群（WaitTemplateGone · Swipe · Scroll 横向 · InputText · StopApp · ClickAt 组合键+多击）· **Phase 4（用户增补）**：WaitWindowGone（等窗口关闭，对称 WaitWindow，配 StopApp 确认真关掉）+ Point 类型手填控件（PointWidget x/y 百分比；Swipe 起止点在 Inspector 手填或连检测节点点输出）。逐 task TDD + 每 task review + 整支 opus 终审，build/vet/全 scope 全绿（除已知 runtime RED 基线）。plan-phase1..4 + 进度账本 .superpowers/sdd/progress.md。**仅剩真机 smoke（用户做）**，全过即移 cold store。
- [work/mcp-node-exec/](work/mcp-node-exec/) — ③ MCP 对外暴露（AI 调我们）：GUI 内置 Streamable HTTP MCP server，暴露 run_node/find_window/写图四件套，design.md + plan.md 就绪。detect-click 完结后的下一主攻（原 focus，本会话未动）。

## Next

1. **detect-click 真机 smoke（用户）** —— Phase 1-3：① 锚点偏移点中 · ⑤ 多命中点最上/第2 · ⑧ 限 ROI · ② 等图消失 · ③ ctrl+点 · ④ 双击 · ⑥ 拖滑块 · ⑦ 搜索框打字 · ⑨ 横向滚 · ⑩ 杀进程；**Phase 4**：WaitWindowGone（开记事本→等其窗口关）· Swipe 在 Inspector 手填 x/y 拖拽 · Swipe 连 ClickTemplate 的点拖拽。全过 → 移 work/detect-click-config 到 cold store。
2. 之后：按 [work/mcp-node-exec/plan.md](work/mcp-node-exec/plan.md) 起 MCP 实现（winutil.EnumTopWindows → ContainerRunner.ExecOutputs 访问器 → internal/services/mcpserver 包 → settings arm 开关 → main.go HTTP server 生命周期 → 设置页 MCP tab → 退役 cmd/yotta-mcp）。

## Open questions

- **detect-click 终审两个已知局限（已落 knowledge，未修、等 demand）**：① Swipe 在 sendinput 后端实际走 PostMessage、读 RawInput 的游戏收不到拖拽 → [knowledge/input/sendinput-drag-uses-postmessage.md](knowledge/input/sendinput-drag-uses-postmessage.md)；② pkg/input 的 SendInput 原语不查注入数、失败上报不到节点层（InputService.error 在 sendinput 后端恒 nil）→ [knowledge/input/sendinput-primitives-ignore-failure.md](knowledge/input/sendinput-primitives-ignore-failure.md)。皆全包/共享原语既有模式、非 Phase 3 回归，整支终审裁定 ship-as-is + 名状在案。
- **预存失败基线**（跑测试判红按此排除）：runtime 缺 fish fixture（apply_direction.json / watchdog_check.json）→ `TestApplyDirection_*` / `TestWatchdog_*` / `TestFishingV2Main_StateCycleSmoke` 恒红；i18n residue / `pnpm lint` 18，见 [knowledge/build/build.md](knowledge/build/build.md)。
- **MCP/AI（③）挂账**：A6/C7 polish 余项（AI/Image 节点分组 · Model combobox · 删连接确认弹窗 FE · SaveImage/LoadImage 编辑期路径校验）+ AI 配置 ① 待 smoke 项（Anthropic 原生 / 本地 Ollama / 删默认清空 / 重启持久化），细节在 ① 配置 spec §9（cold store `archive/specs/2026-06-23-local-ai-config.md`）。AI 系统知识 → [knowledge/nodes/ai-nodes.md](knowledge/nodes/ai-nodes.md) + [knowledge/nodes/held-exec-outputs.md](knowledge/nodes/held-exec-outputs.md)。
- **积压路由**：编辑器 footgun / i18n residue 清理等零散项在 cold store `ideas/`（editor-footgun-backlog.md / misc-tools-backlog.md）。
