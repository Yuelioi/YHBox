---
slice: "41"
title: 运行工作台与调试反馈
status: completed
---

# Slice 41：运行工作台与调试反馈

## Outcome / Question

把 Logs、Timeline、Debug 变成同一 Run Event 事实的独立有界投影，取消普通运行抢焦点，并让单步状态无需猜测。

## Completion criterion

- 编辑器只有一个可折叠/可调高度底部工作台，含 Logs/Timeline/Debug tabs。
- 普通 Run 不自动打开；失败开 Logs；pause/break 开 Debug；Timeline 手动或显式偏好打开。
- Timeline RPC 使用有界 page/range 与 total/page/pages；默认只取最新 200 条，单页硬上限 500。
- Debug snapshot 含 runID/sequence、previous/executed、current/will-execute、pause reason；显示产品标签。
- Step 在新 pause snapshot 前 busy；旧 run/sequence 不覆盖新状态。
- 画布区分 current/previous，三节点 fixture 连续单步不再被误解为卡死。

## Verification

- panel opening state machine 组件测试。
- timeline paging/bounds 服务与大 run 测试。
- debug snapshot、monotonic merge 与 exact 三节点 workflow 测试。
- 普通 run、失败、连续 step、长运行真机验收。

## Out of scope

- 不做完整任意 dock、多屏同步或独立 Trace Server。
- 不复制三份底层事件。

## Result

Completed。

- WorkflowRuntimeWorkbench 将 Logs/Timeline/Debug 收到一个可折叠底部面板；普通 Run 保持关闭，运行失败打开 Logs，调试暂停打开 Debug，Timeline 仅手动进入。
- Timeline 服务与前端改为分页读取，默认最新 200 条且单页上限 500，避免把完整 journal 一次渲染到 DOM。
- DebugSnapshot 新增 previous graph/node；UI 明确显示“刚执行”和“即将执行（尚未执行）”，Step 在新 snapshot 前保持 busy。
- exact 三节点 workflow 的 WebView smoke 已验证 debug start/step/restart/stop；服务、session、panel 测试和 task check 通过。
