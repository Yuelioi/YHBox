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

ai-native-design.md 当前是 231 行、已提交且工作树未修改的设计 artifact；包含产品判断、provider/model/prompt、AI nodes、authoring、MCP、trace/eval、agent instructions 与十项完成定义。现有 registry 已明确 ai-generic-fallback-removal completed，但尚未对其余完成定义做逐项证据处置。

## Out of scope

- 在审计 Slice 中直接实现 AI 功能。
- 因文件较大而机械拆 Topic。
- 重复已完成的 provider-native adapter/profile、typed patch 或 MCP 工作。

## Result

Current。质量门禁已恢复，下一步读取 ai-native-design.md 并逐项建立代码、测试与 commit 证据矩阵。