# AI-native 目标设计完成度处置

Status: ready

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

restore-go-quality-gate。先恢复仓库可信门禁，再审计和展开新的实现 frontier。

## Verification

ai-native-design.md 当前是 231 行、已提交且工作树未修改的设计 artifact；包含产品判断、provider/model/prompt、AI nodes、authoring、MCP、trace/eval、agent instructions 与十项完成定义。现有 registry 已明确 ai-generic-fallback-removal completed，但尚未对其余完成定义做逐项证据处置。

## Out of scope

- 在审计 Slice 中直接实现 AI 功能。
- 因文件较大而机械拆 Topic。
- 重复已完成的 provider-native adapter/profile、typed patch 或 MCP 工作。

## Result

Ready，未开始。原文件继续位于 work/major-upgrade-review/ai-native-design.md，并仅在相关 AI Slice 中按条件读取。
