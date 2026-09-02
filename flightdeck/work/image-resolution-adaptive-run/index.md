# 图片录制与分辨率自适应 Run

## Goal

修复图片变体管理中进入录制后只显示模糊遮罩、无法开始截图的问题；在不增加模板匹配热路径开销的
前提下，让 Run 建立阶段根据目标分辨率预选精确图片变体，缺少精确档时一次性生成同比例派生模板，
使仅含 1080p 图片的工作流可以在标准 2K 场景运行。

## Status

Open

## Current

图片变体 Modal 的录制阻断已修复：新增或指定变体重拍会发起截图动作并同步关闭旧 Modal，不再留下覆盖
截图界面的模糊遮罩；真实组件回归覆盖两条路径。

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
