# Cockpit — YHFish

Updated: 2026-06-23 · 月离 · Stage: ② AI 节点 epic + held-output 实装完成，待真机 smoke

Focus: ② AI 节点 epic（文本 + 结构化类型输出 + vision，三阶段全落）→ [spec](specs/2026-06-23-ai-nodes.md)

Pointers: config → rules.md · conventions → ../CLAUDE.md · artifacts → folder INDEXes · history → archive/

## Next

用户真机 smoke（`task build` 后，AI 三场景 + held-output 跨跳，见 ## Pending Review）→ [plan](plans/2026-06-23-ai-nodes.md)。通过则走 landing 归档 spec/plan(×2) + graduate spec→docs，再起 ③ MCP。

## In Progress

<!-- AUTO:inprogress -->
- [2026-06-23-ai-nodes.md](specs/2026-06-23-ai-nodes.md) — AI 功能 epic 第②块: 图里调 LLM。新 AI 节点(选 connection+model、提示词模板 {{Name}} 插值 + 任意个带类型动态输…
- [2026-06-23-ai-nodes.md](plans/2026-06-23-ai-nodes.md) — ② AI 节点三阶段 TDD 实现计划: Phase A 文本节点(ctx.AI() Provider 指纹缓存 + AI 节点 Spec/Run + {{}}…
<!-- /AUTO -->

## Staged (awaiting land)

<!-- AUTO:staged -->
### Done (awaiting land)
- [2026-06-23-held-exec-outputs.md](specs/2026-06-23-held-exec-outputs.md) — 框架数据流改进(UE 式 held output): exec 节点 fire 时自动把出口 Data 字段存进本次运行的输出缓存(nodeID.field→值…
- [2026-06-23-held-exec-outputs.md](plans/2026-06-23-held-exec-outputs.md) — TDD 实现 held exec output 缓存: ContainerRunner.execOutputs(per-run, 键 nodeID.field)…
<!-- /AUTO -->

## Key Context

- **A6/C7 polish 余项**（不阻塞 smoke，记防忘）：专用 AI/Image 节点分组（现 AI→system、Image→detect 组）· Model combobox（现纯文本手填）· 删连接确认弹窗 FE（后端 `container.AINodesUsingConnection` 已就绪）· SaveImage/LoadImage 编辑期路径校验（现仅运行期 guard）。
- **AI epic 衔接细节**：①/②/③ 衔接（model 选择 / vision / Provider 缓存失效 / 删连接节点引用检查）记在 [archive/specs/2026-06-23-local-ai-config.md](archive/specs/2026-06-23-local-ai-config.md) §9。
- **预存失败基线**（跑测试/检查按此判红）：runtime 缺 fish fixture（[build.md](checklists/build.md)）· i18n residue 42（[misc-tools-backlog](specs/misc-tools-backlog.md)）· `pnpm lint` 18 错（oxlint 1.64 新规则，全在 HEAD）。

## Pending Review

- **② AI 节点 epic 全量真机 smoke**（单测/typecheck/i18n parity 全绿，待用户 `task build` 后实测）→ 通过后：landing 归档 spec/plan + graduate spec→docs，再起 ③ MCP。
  - **文本**：拖 `AI` 节点（System 组）→ 连接选已配 DeepSeek → 模型填 `deepseek-chat` → User 写带 `{{名}}` 提示词 + 声明同名动态输入 → 跑出 `Text` 绑变量下游可读。
  - **结构化**：在 AI 节点「输出口」声明带类型字段（如 `Count` Integer、`Label` String）→ 跑后各字段逐个绑变量、类型正确（非整 → Fail）。
  - **vision**：`Capture`（detect 组，选 JPEG）→ exec 接 AI.In + 数据线接 Capture.Image 到 AI 图像输入 → AI 识图出结果；`Capture → SaveImage` 写盘扩展名对；`LoadImage` 读盘喂 AI。
- **held exec output 框架改进**（Go 单测全绿：跨跳/fan-out/稀疏/子图/loop/未fire 覆盖；FE 类型/parity 绿；删单跳 applyExecDataEdges + EXEC_DATA_NOT_ADJACENT 警告）→ 真机：编辑器里把 AI 结构化输出 `red`/`white` 数据线**跨跳直连**多个/远处节点（免 GetVar、免紧邻），跑出值正确 → 通过后随 AI epic 一并 landing + graduate（spec [§11 实测结论](specs/2026-06-23-held-exec-outputs.md)）。
- **AI 配置 ① 余项 smoke**（核心 DeepSeek 测连接用户已实测）：Anthropic 原生连接 / 本地 Ollama / UI 删默认清空 / 重启持久化 —— 均有单测背书，待用户顺手点。

## Hanging Tasks

- [ ] 无阻塞待办。（积压路由不变：编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md)；bindings / 测试 fixture / AlwaysOnTop / 通道 B smoke → [build.md](checklists/build.md)；i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。）
