# AI Run 诊断与工作流修复计划

## Wayfinding

- [x] 决定 [运行证据与职责边界](slices/run-evidence-boundaries.md)：Timeline、Diagnostic、Log 与附件分别保存什么。
- [x] 决定编辑器定位与重新加载语义：Run 的 Source revision、Graph path、Node ID 如何跳转，脏编辑如何处理。
- [x] 决定 AI 工具 seam：读取证据/节点帮助、提出补丁、验证与应用修改分别由哪个深 module 提供。
- [x] 决定模板匹配诊断策略：最高分等有界聚合、阈值语义，以及派生模板是否需要独立策略。
- [x] 完成 [全节点结果与错误契约审计](slices/node-result-error-audit.md)，决定 Error Envelope 与本地化规则。

## Delivery

- [x] Stage 1：丰富 Run Evidence——来源 revision、模板最佳分/阈值/候选位置与可读状态说明。
- [x] Stage 0：错误契约基础——稳定 error ID、typed params、内部 cause、注册表/schema 与翻译门禁。
- [x] Stage 0B：逐节点 outcome 审计——为 timeout/not-found/exhausted/failed 定义结构化证据和迁移矩阵。
- [x] Stage 2：时间线导航——节点条目一键定位、历史 revision 提示、失败/timeout 摘要与筛选。
- [x] Stage 3：Run MCP——有界 Run list/get/evidence/diagnose 输入，并补齐节点帮助的稳定可读投影。
- [x] Stage 4：Workflow Repair——AI 生成 typed patch 提案、编译验证、差异审查、CAS 应用与编辑器安全 reload。
- [x] Stage 5：诊断 AI 产品入口——设置诊断用途 profile、工作流内诊断面板、对话定位与建议应用。
- [x] Stage 6：日志职责——补足 adapter/infrastructure 日志、Run attribution 和支持包导出，不复制 Timeline。
- [x] Stage 7：阈值研究——保留单一默认阈值 0.85；用 Run Evidence 暴露派生模板实际最高分，不加入未经负样本验证的全局 0.75 补偿。

## Decisions

- 第一版不自动持久化逐帧截图；Timeline 保存有界数值证据，显式诊断脚本保存截图附件。
- AI Repair 必须显示候选并由用户接受，继续复用 typed patch、编译预检和 exact revision CAS。
- 诊断 profile 是 AI profile 的用途引用，不复制模型配置或凭据。
