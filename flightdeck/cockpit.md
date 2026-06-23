# Cockpit — YHFish

**Last updated**: 2026-06-23 by 月离 (AI 功能 epic ① 本地 AI 配置落地 —— llm 包(OpenAI/Anthropic 双官方 SDK + Provider 接口 + 连接池)+ AIService 测连接 RPC + SettingsAI 设置 tab; 8 commits @ feat/v2-foundation; 核心 DeepSeek 测连接真机已过, spec/plan 归档。**默认不 push**)。
**Active focus**: **AI 功能 epic** —— ① 本地 AI 配置已落(实现 + 核心真机验)。下一步 ② AI 节点(图里调 LLM, 复用 llm.Provider + 加 vision)。**默认不 push**。

## 进行中

<!-- AUTO:inprogress -->
- [2026-06-23-ai-nodes.md](specs/2026-06-23-ai-nodes.md) — AI 功能 epic 第②块: 图里调 LLM。新 AI 节点(选 connection+model、提示词模板 {{Name}} 插值 + 任意个带类型动态输…
- [2026-06-23-ai-nodes.md](plans/2026-06-23-ai-nodes.md) — ② AI 节点三阶段 TDD 实现计划: Phase A 文本节点(ctx.AI() Provider 指纹缓存 + AI 节点 Spec/Run + {{}}…
<!-- /AUTO -->

## 下一步

- **AI 功能 epic 续做**(① 已落): **② AI 节点**(图里调 LLM —— 节点选 connectionID+model、动态带类型 IO、structured output、**vision 图像输入**、Provider 缓存+失效+池调优); **③ MCP 对外暴露**(已有 `cmd/yotta-mcp` spike, 缺执行容器/节点工具)。②/③ 衔接细节见 [archive spec §9](archive/specs/2026-06-23-local-ai-config.md)。
- 其它候选池: 临时窗口抓取(EnumWindows 选窗截图); 复发#5 promotion(前台容器全局指针升 checklist); cv-perception 池剩余 ([cv-perception](specs/cv-perception-pool.md)); idea 池([editor-footgun](specs/editor-footgun-backlog.md) · [misc-tools](specs/misc-tools-backlog.md))。

## 待复核

- 无。

## 待验证

- **AI 配置 ① 余项真机 smoke**(核心 DeepSeek 测连接用户已实测过): Anthropic 原生连接 / 本地 Ollama / UI 删默认清空 / 重启持久化 —— 均有单测背书, 待用户顺手点。源在归档 plan 的 verify 字段(preflight `--verify-pending` 自动提示)。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- **知识在哪**: 常驻技术知识/系统架构 → [docs/](docs/INDEX.md); 加节点/写代码/UI 的规范与收尾清单 → [checklists/](checklists/INDEX.md)(按 when_to_read 路由); 反复踩的坑 → [incidents/](incidents/INDEX.md); 历史设计/执行记录 → `archive/specs|plans/` 按日期。
- **AI 功能 epic 进度**: ① 本地 AI 配置已落(`internal/services/llm` Provider+双官方 SDK adapter+连接池; `Settings.AI` 连接; `AIService.TestConnection`; SettingsAI tab)。②/③ 衔接(model 选择、vision、Provider 缓存/失效、删连接节点引用检查)记在 [archive/specs/2026-06-23-local-ai-config.md](archive/specs/2026-06-23-local-ai-config.md) §9。
- **已知预存失败(非回归, 跑测试/检查时按此判红)**: runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **42**(misc-tools-backlog 未翻译 UI: SettingsLauncher/FloatingLauncherView/HudShell/IconPicker + 1 处 console.log; 另 11 处 editorTheme.ts 查找面板中文 = 有意 zh 映射, 别翻); `pnpm lint` 预存 **18** 错(oxlint 1.64 新规则, 已实证全在 HEAD)。
