---
slice: "41"
title: 运行工作台与调试反馈
status: in_progress
---

# Slice 41：运行工作台与调试反馈

## Outcome / Question

让 Debug Start、Step、Continue、Pause 与 Stop 在 RPC/event 任意到达顺序下保持单调一致，并将调试器改成以“当前状态与下一步动作”为核心的桌面工作台。

## Completion criterion

- Debug Start 期间到达的同 runID 新 generation snapshot 不会因 activeRun 尚未赋值而丢失；旧 RPC 响应不能覆盖新 event。
- 启动调试稳定显示 paused 和 Step/Continue；Step busy 直到同 runID 更高 generation；旧 run/旧 generation 被拒绝。
- paused 首层明确“将在执行哪个节点前暂停”、pause reason 和主要控制动作；previous/current/next 使用产品标签。
- running、paused、completed、canceled/failed 的文案和内容结构分别正确；终态不显示“即将执行（尚未执行）”。
- Inputs/Outputs/Run State/Queue 只在有内容时显示数量和内容，通过 tabs/折叠渐进披露，不平铺四列“无”。
- 内部 graph path、run ID、digest 退到详情/复制层，不抢占主任务。
- Logs、Timeline、Debug 继续共享一个可折叠底部工作台；普通 Run 不抢焦点，Timeline 继续有界分页。

## Verification

- EditorSession RPC-before-event、event-before-RPC、旧 generation、旧 run 回归测试。
- WorkflowDebuggerPanel paused/running/terminal、空数据、Step 控制和 node focus 组件测试。
- exact 三节点 workflow 连续 Start/Step/Continue/Stop WebView smoke。
- 阶段末与 Slice 39 一起执行 task check、production build、人工截图和提权真机验收。

## Out of scope

- 不做完整任意 dock、多屏同步或独立 Trace Server。
- 不复制 Logs/Timeline/Debug 底层事件。
- 不把 backend snapshot 直接作为最终 UI 信息架构。

## Result

Implementation complete; awaiting user acceptance。EditorSession 在 StartDebugRun RPC 返回前缓存同 runID 的早到 snapshot，并按 generation 单调合并；精确红测已覆盖 event-before-RPC，不再由旧 RPC running 响应覆盖 paused event。Debugger 首层现在按 paused/running/terminal 提供正确状态、控制和真实执行位置；空快照渐进披露，内部路径退到技术详情，终态不再显示“即将执行”或空 previous/next 占位。真实组件测试、`task check`、production build、WebView smoke 与人工终态截图检查均通过；尚需用户用原三节点 workflow 验收 Start/Step/Continue。
