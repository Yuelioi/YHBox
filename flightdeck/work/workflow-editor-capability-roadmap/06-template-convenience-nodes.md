# Slice 6：模板等待与点击复合节点

## Outcome / Question

恢复 WaitTemplate、WaitTemplateGone、ClickTemplate 的表达力，同时保持显式 target、BlobRef、timeout 和错误语义。

## Completion criterion

- 按真实使用频率确定顺序，交付等待出现、等待消失、匹配后点击的明确 contract。
- 声明 exact target、BlobRef、阈值、poll/timeout、result/error。
- 匹配与点击绑定同一 target/session 和坐标变换，不读取环境 HWND。
- 通过 capability admission 和现有 automation adapter；日志只记模板摘要/结果。
- 目录、inspector、compiler、timeline 全部使用新 contract。
- 失败可解释，避免用户手拼 capture/match/retry/delay。

## Blocked by

Stage 2；实现前读取 knowledge/nodes/add-node.md。

## Verification

定向 contract/compiler/runtime；与 Slice 7 用真实模板流程批量验收。

## Out of scope

任意视觉脚本、OCR 全家桶、环境窗口回退和旧 JSON 捷径。

## Result

Planned。值得恢复，但必须重做。
