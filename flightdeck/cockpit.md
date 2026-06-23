# Cockpit — YHFish

**Last updated**: 2026-06-23 by 月离 (② AI 节点 epic **三阶段 A+B+C 全部实装**, 16 feat commits @ feat/v2-foundation。文本调用 + 结构化带类型输出 + vision 识图端到端;新框架机制 DynamicDataFields + applyExecDataEdges 扩动态输入;node.Image 一等流动值 + Capture/SaveImage/LoadImage(删 Screenshot 切净)。全量 go test 仅剩预存 fish fixture 基线 + 1 flaky 计时测试, 无回归;FE typecheck/i18n parity 绿。**待真机 smoke。默认不 push**)。
**Active focus**: **② AI 节点 epic — 实装完成, 待真机 smoke**。三阶段全落: A 文本节点+Provider 缓存 · B 结构化类型输出(ChatStructured 双 SDK 三模式 + DynamicDataFields)· C vision(node.Image + Capture/SaveImage/LoadImage + 多模态 + exec-data 直连)。**下一步: 用户真机 smoke(task build 后)→ 通过则 landing 归档 spec/plan + graduate spec→docs → ③ MCP**。**默认不 push**。

## 进行中

<!-- AUTO:inprogress -->
- [2026-06-23-ai-nodes.md](specs/2026-06-23-ai-nodes.md) — AI 功能 epic 第②块: 图里调 LLM。新 AI 节点(选 connection+model、提示词模板 {{Name}} 插值 + 任意个带类型动态输…
- [2026-06-23-ai-nodes.md](plans/2026-06-23-ai-nodes.md) — ② AI 节点三阶段 TDD 实现计划: Phase A 文本节点(ctx.AI() Provider 指纹缓存 + AI 节点 Spec/Run + {{}}…
<!-- /AUTO -->

## 下一步

- **② AI 节点 epic 实装完成** —— 下一步**用户真机 smoke**(`task build` 后,见「待验证」)。通过则走 landing: 归档 spec/plan + graduate spec→docs([2026-06-23-ai-nodes](specs/2026-06-23-ai-nodes.md) 带 graduate:true)。
- **A6/C7 polish 余项**(不阻塞 smoke,记防忘): 专用 AI/Image 节点分组(现 AI→system、Image→detect 组)· Model combobox(现纯文本手填模型名)· 删连接确认弹窗 FE(后端 `container.AINodesUsingConnection` 已就绪)· SaveImage/LoadImage 编辑期路径校验(现仅运行期 guard)。
- **③ MCP 对外暴露**(②后): 已有 `cmd/yotta-mcp` spike, 缺执行容器/节点工具。
- 其它候选池: cv-perception 池剩余 ([cv-perception](specs/cv-perception-pool.md)); idea 池([editor-footgun](specs/editor-footgun-backlog.md) · [misc-tools](specs/misc-tools-backlog.md))。

## 待复核

- 无。

## 待验证

- **② AI 节点 epic 全量真机 smoke**(用户将做,`task build` 后;单测/typecheck/i18n parity 全绿):
  - **文本**: 拖 `AI` 节点(System 组)→ 连接下拉选已配 DeepSeek → 模型填 `deepseek-chat` → User 写带 `{{名}}` 提示词 + 声明同名动态输入 → 跑出 `Text` 绑变量下游可读。
  - **结构化**: 在 AI 节点「输出口」声明带类型字段(如 `Count` Integer、`Label` String)→ 跑后各字段逐个绑变量、类型正确(非整 → Fail)。
  - **vision**: `Capture`(detect 组,选 JPEG)→ exec 接 AI.In + 数据线接 Capture.Image 到 AI 的图像输入 → AI 识图出结果;`Capture → SaveImage` 写盘扩展名对;`LoadImage` 读盘喂 AI。
- **AI 配置 ① 余项真机 smoke**(核心 DeepSeek 测连接用户已实测过): Anthropic 原生连接 / 本地 Ollama / UI 删默认清空 / 重启持久化 —— 均有单测背书, 待用户顺手点。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- **知识在哪**: 常驻技术知识/系统架构 → [docs/](docs/INDEX.md); 加节点/写代码/UI 的规范与收尾清单 → [checklists/](checklists/INDEX.md)(按 when_to_read 路由); 反复踩的坑 → [incidents/](incidents/INDEX.md); 历史设计/执行记录 → `archive/specs|plans/` 按日期。
- **AI 功能 epic 进度**: ① 本地 AI 配置已落(`internal/services/llm` Provider+双官方 SDK adapter+连接池; `Settings.AI` 连接; `AIService.TestConnection`; SettingsAI tab)。②/③ 衔接(model 选择、vision、Provider 缓存/失效、删连接节点引用检查)记在 [archive/specs/2026-06-23-local-ai-config.md](archive/specs/2026-06-23-local-ai-config.md) §9。
- **已知预存失败(非回归, 跑测试/检查时按此判红)**: runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **42**(misc-tools-backlog 未翻译 UI: SettingsLauncher/FloatingLauncherView/HudShell/IconPicker + 1 处 console.log; 另 11 处 editorTheme.ts 查找面板中文 = 有意 zh 映射, 别翻); `pnpm lint` 预存 **18** 错(oxlint 1.64 新规则, 已实证全在 HEAD)。
