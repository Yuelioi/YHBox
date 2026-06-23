# Cockpit — YHFish

Updated: 2026-06-23 · 月离 · Stage: ② AI epic + held-output 已 land（真机 smoke 过、graduate→docs）；③ MCP 待起

Focus: ③ MCP 对外暴露（AI 调我们）—— 待起 spec（已有 `cmd/yotta-mcp` spike）

Pointers: config → rules.md · conventions → ../CLAUDE.md · artifacts → folder INDEXes · history → archive/

## Next

起草 ③ MCP 对外暴露 spec：给 AI 提供执行容器/节点的工具（已有 `cmd/yotta-mcp` spike，缺执行容器/节点能力）。

## In Progress

<!-- AUTO:inprogress -->
- [2026-06-23-mcp-node-exec.md](specs/2026-06-23-mcp-node-exec.md) — AI 功能 epic 第③块(MCP 对外暴露 / AI 调我们): GUI 内置 Streamable HTTP MCP server, 暴露通用 run_n…
- [2026-06-23-mcp-node-exec.md](plans/2026-06-23-mcp-node-exec.md) — ③ MCP 节点执行实现计划: winutil.EnumTopWindows + ContainerRunner.ExecOutputs 访问器 + inter…
<!-- /AUTO -->

## Staged (awaiting land)

<!-- AUTO:staged -->

<!-- /AUTO -->

## Key Context

- **A6/C7 polish 余项**（不阻塞 smoke，记防忘）：专用 AI/Image 节点分组（现 AI→system、Image→detect 组）· Model combobox（现纯文本手填）· 删连接确认弹窗 FE（后端 `container.AINodesUsingConnection` 已就绪）· SaveImage/LoadImage 编辑期路径校验（现仅运行期 guard）。
- **AI 系统知识**（②已 land）→ [docs/ai-nodes.md](docs/ai-nodes.md) + [docs/held-exec-outputs.md](docs/held-exec-outputs.md)；①/③ 衔接细节在 [archive/specs/2026-06-23-local-ai-config.md](archive/specs/2026-06-23-local-ai-config.md) §9。
- **预存失败基线**（跑测试/检查按此判红）：runtime 缺 fish fixture（[build.md](checklists/build.md)）· i18n residue 42（[misc-tools-backlog](specs/misc-tools-backlog.md)）· `pnpm lint` 18 错（oxlint 1.64 新规则，全在 HEAD）。

## Pending Review

- **AI 配置 ① 余项 smoke**（核心 DeepSeek 测连接用户已实测）：Anthropic 原生连接 / 本地 Ollama / UI 删默认清空 / 重启持久化 —— 均有单测背书，待用户顺手点。

## Hanging Tasks

- [ ] 无阻塞待办。（积压路由不变：编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md)；bindings / 测试 fixture / AlwaysOnTop / 通道 B smoke → [build.md](checklists/build.md)；i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。）
