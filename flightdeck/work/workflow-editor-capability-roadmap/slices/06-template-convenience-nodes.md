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

Completed。按 3.1 contract 重做，没有复活旧 container/runtime、Template GUID 或 ambient HWND。

- 新增 WaitTemplate、ClickTemplate、WaitTemplateGone；template 是持久 Image BlobRef，region/threshold/timeout/poll 明确，结果给出 matched/score/center/bounds，并分别路由 found/completed/gone、timeout、failed。
- capture-target 与 input-target 通过同一 config slot 解析同一 exact target；ClickTemplate 在 settle 后重新捕获定位，把像素中心转换为该窗口 ratio 坐标再点击。
- 共用 poll 模块每次读取新 capture，0ms 为单帧，最后一段等待截短到 timeout，所有等待与资源调用都接受 Run cancellation。
- Blob 只经受限 blob-read capability 读取；捕获帧不写临时资产，journal 只记录动作名和有界 counters。
- 定向验证覆盖 contract/authoring projection、appears/disappears/timeout/cancel 轮询，以及真实 admission、同 target capture+match+click；contracts、Go vet/staticcheck、frontend typecheck/i18n 通过。
- 源码提交：472931d0。Stage 3 全量门禁留到 Slice 7 完成后统一执行。
