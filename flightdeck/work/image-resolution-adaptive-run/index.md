# 图片录制与分辨率自适应 Run

## Goal

修复图片变体管理中进入录制后只显示模糊遮罩、无法开始截图的问题；在不增加模板匹配热路径开销的
前提下，让 Run 建立阶段根据目标分辨率预选精确图片变体，缺少精确档时一次性生成同比例派生模板，
使仅含 1080p 图片的工作流可以在标准 2K 场景运行。

## Status

Open

## Current

图片变体 Modal 的录制阻断已修复：新增或指定变体重拍会发起截图动作并同步关闭旧 Modal，不再留下覆盖
截图界面的模糊遮罩；真实组件回归覆盖两条路径。工作流内重录返回的图片变体经过 Wails/Vue event 边界时
可能成为 reactive projection，旧代码对其调用 `structuredClone` 会抛 `DataCloneError`，截图成功后仍显示
“发生未知错误”；现改为按 portable JSON 边界复制，并为无图片结果、过期目标版本和写入失败提供独立恢复文案。
相邻录制链也已重新审计：Finalize/Stop/倒计时启动、pending event、模板资产与窗口捕获不再通过字符串事件或
原始 Go error 穿越 Wails；所有 Problem 显示 operationId，并以同一 ID 在本地日志记录内部 cause。架构测试禁止
service event 重新加入 `{"error": "..."}` 旧协议。

Run 分辨率自适应已接入生产 composition：Application 在编译/Admission 前持有本次 Run 的 configured-target
generation，通过 target snapshot 一次读取被图片资源引用的 slot 分辨率。精确变体直接成为 Compiler 的
Run override；缺档且宽高比相容时，`internal/runprepare` 读取最近变体、一次缩放为目标尺寸并写入 CAS，
Compiler 保留原 SourceHash 但让 Program 固定消费派生 Blob。派生物由 Run retention 持有，内存编码缓存限制为
64 MiB/256 项；逐帧模板匹配路径没有新增选档、缩放、map 或 mutex 查询。

定向组件、Compiler override、target describe、Planner 缓存/比例保护和相关 Application/Bootstrap 测试均通过；
Planner race、Impeccable 检测和仓库 `task check` 通过。真实 2K“异环 看电影”已证明 1080p→2K 派生链路
生效：失败节点消费与现场诊断一致的 2934-byte、120×33 派生模板；但小型文字模板“弗罗斯特”在 2K
重新栅格化后比缩放模板更锐利，最高匹配分 0.824802，低于默认阈值 0.85，因此超时。已新增只读真机
模板诊断脚本保存当前帧、原模板、派生模板和分数证据；隔离 WebView smoke 的空 fixture 问题仍待处理。

## Next

1. 用真机诊断脚本比较一组可匹配/不可匹配模板，决定小型文字模板的恢复策略并建立回归。
2. 用同时含 1080p/2K 的资源复测精确 2K 优先，并检查重复 Run 不再执行图片 resize。
3. 修复或绕开隔离 smoke fixture 的空工作流问题，完成 [实施计划](plan.md) 的桌面截图验收。

## Progress

- 2026-09-03 修复录制准备偶发“目标不可用”：旧 `AcquireRecordingTarget` 先用 `Execute(activate)` 解析窗口，随后
  `ResolveTarget` 再枚举一次，同一动作存在两次匹配竞态。Win32 adapter 现于同一 driver gate 内只解析一次，
  用该 HWND 完成置前并原样返回；准备和倒计时结束后的再次激活均复用该原子 seam。倒计时文案明确无需点击目标。
- 2026-09-03 录制倒计时结束后、安装原生采集 hook 前，再通过同一 configured target adapter 激活并解析目标；
  仅当 HWND 与准备阶段锁定的窗口完全一致才开始录制。目标消失或身份变化时回到 idle、释放两份 generation
  lease，并返回带 operationId 的 `recording.start.target_reactivation_failed`，避免用户为确认焦点点击游戏。
- 2026-09-03 修复 `build:frontend` 的陈旧 production bundle：旧 `status` 只比较 dev/production marker，导致同模式下
  即使 `frontend/src/**` 更新也跳过构建，Go EXE 随后嵌入旧 UI。新增统一输入时间戳检查和回归测试；当前 dist
  已确认不再包含录制 HUD 旧副标题。
- 2026-09-03 精简录制 HUD：移除重复副标题，暂停/继续/完成/取消按钮以内联 9px 辅助文字展示热键；新增可配置
  且由 LL hook 拦截的 F7“取消并丢弃录制”。HUD 订阅 `hotkey:changed`，录制中的实际 hook 与显示同步热更新；
  标题栏增加置顶/优先使用快捷键状态图标。相关 bindings、Go/前端测试与 `task check` 通过。
- 2026-09-03 根据 owner 真实重录失败建立 reactive event payload 回归：修复 `structuredClone` 无法复制 proxy
  导致的截图后写入失败，并把三条本地失败从 `UNKNOWN_ERROR` 拆为可操作反馈；前端定向测试与 `task check`
  通过。
- 2026-09-03 根据真实“录制保存失败 / transport.unstructured_failure”重新审计录制与捕获错误链：异步事件改传
  canonical Problem，录制开始/停止/Finalize/pending、模板创建/重拍与窗口捕获均补稳定 ID、恢复动作和
  operationId→日志 cause 关联；增加禁止字符串错误事件的架构门禁。
- 2026-09-02 建立 Work；确认“编译/预渲染”不是用户应理解的底层术语，产品入口暂定评估为“优化运行”。
- 2026-09-02 已有未提交 UI 改动：共享图片变体管理 Modal、指定变体重拍、最后一档按动作拦截；前端
  `task check` 通过。隔离 WebView smoke 因 fixture 未生成工作流行而无法完成截图验收。
- 2026-09-02 用组件红绿回归修复双 Modal 遮罩阻断；新增 Run 图片准备深 module、target snapshot 描述 seam
  和 Compiler resource override，使精确选档或同比例一次缩放发生在 Run 建立阶段。33 个受影响 Go package、
  前端 495 项测试、Planner race、文档、UI 检测、正式 Windows build 与桌面启动 smoke 通过。
- 2026-09-02 新增真机模板诊断入口并在真实 2560×1440“异环”画面复现指定资源：原 90×25 模板得分
  0.280410，派生 120×33 模板得分 0.824802，低于 0.85；确认派生生效且失配来自小型文字在目标分辨率
  重新栅格化后的像素差异，而非选档或目标解析失败。

## References

- [稳定上下文](context.md) — 产品语义、运行策略和性能边界。
- [实施计划](plan.md) — 修复录制阻断并交付 Run 分辨率自适应的阶段清单。
