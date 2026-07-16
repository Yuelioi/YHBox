# AI-native 目标设计完成度处置

Status: current

## Question

ai-native-design.md 的完成定义中，哪些 outcome 已被当前 3.1 代码、测试和提交满足，哪些仍是独立剩余工作，哪些已因后续架构决策过时？

## Completion criterion

- 逐项审计 ai-native-design.md 的产品判断、模块边界和完成定义，引用具体代码、测试、commit 或缺口。
- 把每项处置为 completed、remaining 或 obsolete；不得把已完成的 provider-native 工作重新包装成待办。
- 只有 destination 不变且 outcome/verification 可精确陈述的 remaining 项才成为 major-upgrade-review sibling Slice。
- destination 若变为 3.1 发布后的独立 AI 平台演进，才提出新 Topic，并说明独立完成条件。
- 更新 slices/map.md 与相关 Read if 路由；设计 artifact 本身继续保留，不改造成 giant Slice。
- 审计结果 checkpoint 并独立 commit。

## Blocked by

无。restore-go-quality-gate 已由 27e01b17 完成，完整 task check 为绿。

## Verification

- `ab1f4cf4` 建立 OpenAI Responses / Anthropic Messages 原生 adapter、完整 Outcome/usage/request identity 与 strict JSON Schema；`99c3f5ff` 删除 generic Chat 与 prompt JSON fallback。
- `4b630f70` 建立 slot-bound Generate/Extract 节点；`7e9fb87c` 建立 content-addressed ModelProfile、installation snapshot、OS credential binding 与 workflow consent。
- `b4aa17aa` 建立 UI/MCP 共用 typed patch、strict Compiler、catalog search/describe、schema-validated structuredContent；desktop composition 不注册 MCP listener。
- 当前 `internal/nodes31runtime/ai.go` 将节点 config 的任意 `instructions` 直接映射为 OpenAI `instructions` / Anthropic `system`，因此 dynamic/untrusted data 不进入高权限 block 尚未满足。
- ModelProfile 和 strict Schema 已可 hash；仓库没有 PromptManifest、ToolManifest/ToolSet、Agent node/runtime、eval runner/gate 或 AI authoring loop。EvaluationSuite 目前只是 profile metadata。
- Run AdapterAction 已记录 provider/request/usage 摘要，EditorSession 已显示 compiler diagnostics，MCP preview 已返回 capability plan；但没有 prompt/tool/schema lineage、AI patch diff/permission delta 审阅或完整脱敏 AI trace。
- tracked `AGENTS.md` 已满足 contributor agent canonical contract；设计中的固定 `internal/ai/*` 目录拓扑只是实现草图，不应为目录对齐机械拆包。
- 完整处置矩阵位于 `work/major-upgrade-review/research/ai-native-disposition-2026-07-17.md`。

## Out of scope

- 在审计 Slice 中直接实现 AI 功能。
- 因文件较大而机械拆 Topic。
- 重复已完成的 provider-native adapter/profile、typed patch 或 MCP 工作。

## Result

证据审计完成，等待独立提交与 handoff。九条完成定义中 1、2、7 已 completed；3、4、8 remaining；5 的 strict Extract completed、Agent remaining；6 的 shared typed substrate completed、AI authoring client remaining；9 的基础 diagnostics/usage trace completed、change review 与完整 provenance remaining。Session resource 按原设计保留为需求驱动的 post-3.1 选项，固定包目录布局按 obsolete implementation sketch 处置。