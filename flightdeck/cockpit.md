# Cockpit — YHFish

Focus: ③ MCP 对外暴露（AI 调我们）—— spec + plan 已就绪，待按 plan 起实现（已有 `cmd/yotta-mcp` spike）。

## In flight

- [work/mcp-node-exec/](work/mcp-node-exec/) — AI 功能 epic 第③块（MCP 对外暴露 / AI 调我们）：GUI 内置 Streamable HTTP MCP server，暴露通用 `run_node`（单动作节点探测，复用 held-output 缓存收割输出，含 Capture 图像）+ `find_window` + 写图四件套（后端换 GUI 真实 store），闭合「AI 跑节点实验 → save_container 生成容器」环；全局 arm 安全开关默认关。design.md + plan.md 都在该 folder。

## Next

按 [work/mcp-node-exec/plan.md](work/mcp-node-exec/plan.md) 起实现：winutil.EnumTopWindows → ContainerRunner.ExecOutputs 访问器 → `internal/services/mcpserver` 包（run_node harness + find_window/list_windows + authoring 工具迁移）→ settings arm 开关 → main.go Streamable HTTP server 生命周期 → 设置页 MCP tab → 退役 cmd/yotta-mcp。TDD，mock window/input/capture fixture。

## Open questions

- **A6/C7 polish 余项**（不阻塞 smoke，记防忘）：专用 AI/Image 节点分组（现 AI→system、Image→detect 组）· Model combobox（现纯文本手填）· 删连接确认弹窗 FE（后端 `container.AINodesUsingConnection` 已就绪）· SaveImage/LoadImage 编辑期路径校验（现仅运行期 guard）。
- **AI 系统知识**（②已 land）→ [knowledge/nodes/ai-nodes.md](knowledge/nodes/ai-nodes.md) + [knowledge/nodes/held-exec-outputs.md](knowledge/nodes/held-exec-outputs.md)；①/③ 衔接细节在 ① 配置 spec §9（已归档到 cold store `archive/specs/2026-06-23-local-ai-config.md`，不在 deck 内）。
- **预存失败基线**（跑测试/检查按此判红）：runtime 缺 fish fixture（见 [knowledge/build/build.md](knowledge/build/build.md)）· i18n residue 42 · `pnpm lint` 18 错（oxlint 1.64 新规则，全在 HEAD）。
- **AI 配置 ① 余项待 smoke**（核心 DeepSeek 测连接用户已实测，均有单测背书）：Anthropic 原生连接 / 本地 Ollama / UI 删默认清空 / 重启持久化 —— 待用户顺手点。同列 ③ 的真机 smoke（见 plan.md 收尾，未武装拒动 / 武装后 Capture·ClickAt·save_container / GUI 跑时 BUSY）。
- **积压路由**（无阻塞待办）：编辑器 footgun、i18n residue 清理等零散项已挪到 cold store `ideas/`（`editor-footgun-backlog.md` / `misc-tools-backlog.md`）；bindings / 测试 fixture / AlwaysOnTop / 通道 B smoke 见 [knowledge/build/build.md](knowledge/build/build.md)。
