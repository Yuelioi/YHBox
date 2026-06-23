# Cockpit — YHFish

**Last updated**: 2026-06-23 by 月离 (② AI 节点 —— **Phase A 整块完成**(A1-A6: ctx.AI() 指纹缓存 + AI 文本节点 Spec/Run/{{}} 模板 + validateAI + 删连接引用扫描 + FE 连接下拉/i18n)**+ B1 DynamicDataFields 框架机制**,7 feat commits 全单测绿 + FE parity/typecheck 绿 @ feat/v2-foundation。文本 AI 节点端到端可真机 smoke。**默认不 push**)。
**Active focus**: **② AI 节点**(图里调 LLM)—— **Phase A 完成 + B1 已落**。文本节点可用(拖节点、连接下拉选、{{}} 提示词、跑出 Text)。**resume 点: Phase B 结构化输出 B2(OpenAI json_schema native, go doc 核实 Stainless union)→ B3 Anthropic 强制 tool-use → B4 提示词注入 → B5 AI 节点接结构化 → B6 FE 输出声明编辑器 → Phase C vision**。逐任务见 [plan](plans/2026-06-23-ai-nodes.md)。**默认不 push**。

## 进行中

<!-- AUTO:inprogress -->
- [2026-06-23-ai-nodes.md](specs/2026-06-23-ai-nodes.md) — AI 功能 epic 第②块: 图里调 LLM。新 AI 节点(选 connection+model、提示词模板 {{Name}} 插值 + 任意个带类型动态输…
- [2026-06-23-ai-nodes.md](plans/2026-06-23-ai-nodes.md) — ② AI 节点三阶段 TDD 实现计划: Phase A 文本节点(ctx.AI() Provider 指纹缓存 + AI 节点 Spec/Run + {{}}…
<!-- /AUTO -->

## 下一步

- **② AI 节点续做**(Phase A 完成 + B1 已落): **Phase B 结构化类型输出** —— **B2** `llm.ChatStructured` 接口 + OpenAI `response_format` json_schema native(go doc 核实 `ChatCompletionNewParamsResponseFormatUnion.OfJSONSchema`→`shared.ResponseFormatJSONSchemaParam`,httptest 验线格式)+ `KindUnsupported` → **B3** Anthropic 强制 tool-use native → **B4** 提示词注入模式 + 容错解析 → **B5** AI 节点接结构化(`config.Outputs[]`→schema→ChatStructured→逐字段 Set + Integer coerce + Fail 带 Text)→ **B6** FE 输出声明编辑器(镜像 DynamicInputsEditor)。**Phase C vision**(node.Image + Capture/SaveImage/LoadImage 删 Screenshot + 多模态 + applyExecDataEdges 扩动态输入)。逐任务见 [plan](plans/2026-06-23-ai-nodes.md)。
- **A6 polish 余项**(不阻塞 smoke,记防忘): 专用 AI/Image 节点分组(现 fallback system 组)· Model combobox(现纯文本手填)· 删连接确认弹窗 FE(后端 `AINodesUsingConnection` 已就绪)。
- **③ MCP 对外暴露**(②后): 已有 `cmd/yotta-mcp` spike, 缺执行容器/节点工具。
- 其它候选池: cv-perception 池剩余 ([cv-perception](specs/cv-perception-pool.md)); idea 池([editor-footgun](specs/editor-footgun-backlog.md) · [misc-tools](specs/misc-tools-backlog.md))。
- 其它候选池: 临时窗口抓取(EnumWindows 选窗截图); 复发#5 promotion(前台容器全局指针升 checklist); cv-perception 池剩余 ([cv-perception](specs/cv-perception-pool.md)); idea 池([editor-footgun](specs/editor-footgun-backlog.md) · [misc-tools](specs/misc-tools-backlog.md))。

## 待复核

- 无。

## 待验证

- **② AI 节点 Phase A 真机 smoke**(用户将做): 拖 `AI` 节点(System 组)→ 连接下拉选已配 DeepSeek → 模型填 `deepseek-chat` → User 写带 `{{名}}` 的提示词 + 声明同名动态输入 → 跑出 `Text`,绑变量下游 GetVar 读到。`task build` 后真机。单测/typecheck/i18n parity 已绿。
- **AI 配置 ① 余项真机 smoke**(核心 DeepSeek 测连接用户已实测过): Anthropic 原生连接 / 本地 Ollama / UI 删默认清空 / 重启持久化 —— 均有单测背书, 待用户顺手点。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- **知识在哪**: 常驻技术知识/系统架构 → [docs/](docs/INDEX.md); 加节点/写代码/UI 的规范与收尾清单 → [checklists/](checklists/INDEX.md)(按 when_to_read 路由); 反复踩的坑 → [incidents/](incidents/INDEX.md); 历史设计/执行记录 → `archive/specs|plans/` 按日期。
- **AI 功能 epic 进度**: ① 本地 AI 配置已落(`internal/services/llm` Provider+双官方 SDK adapter+连接池; `Settings.AI` 连接; `AIService.TestConnection`; SettingsAI tab)。②/③ 衔接(model 选择、vision、Provider 缓存/失效、删连接节点引用检查)记在 [archive/specs/2026-06-23-local-ai-config.md](archive/specs/2026-06-23-local-ai-config.md) §9。
- **已知预存失败(非回归, 跑测试/检查时按此判红)**: runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **42**(misc-tools-backlog 未翻译 UI: SettingsLauncher/FloatingLauncherView/HudShell/IconPicker + 1 处 console.log; 另 11 处 editorTheme.ts 查找面板中文 = 有意 zh 映射, 别翻); `pnpm lint` 预存 **18** 错(oxlint 1.64 新规则, 已实证全在 HEAD)。
