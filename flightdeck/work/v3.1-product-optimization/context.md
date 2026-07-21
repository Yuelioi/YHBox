# 3.1 产品创作体验与运行工作台优化 context

## What matters

Yotta 3.1 的唯一产品事实是 Workflow Source、Catalog/Node Contract、Compiler 和统一 runtime。
恢复旧体验只能复用用户心智和已验证交互，不得复制 3.0 Container、registry、localStorage store 或
第二套执行路径。当前已批准的 Stage 全部完成；下一次启动首先应确认用户是否带来新的真机反馈，
而不是自动继续历史清单。

## Decisions

- Selection、execution、debug 与 validation 是独立状态，不复用一种节点高亮表达。
- 复杂节点使用通用 Authoring Projection 加类型级 Editor Adapter；画布节点保持低密度，完整参数在 Inspector。
- Macro 与精准 InputClip 在领域、资源、编辑器和回放节点上分轨，不隐式互转。
- Tab 搜索和左侧目录共享 Catalog Projection；Snippet 快捷键必须可见、可校验冲突且只在编辑器上下文生效。
- 离开脏工作流必须提供取消、放弃、保存并退出三路选择，保存失败不得继续导航。
- 单对象短流程使用 Modal 保留列表上下文；长生命周期、多页面任务才使用独立路由。
- Stage 内运行最小定向门禁，Stage 完成后再统一执行 `task check` 和触发的真实宿主 smoke。
- 精准 InputClip 固化录制源 `counts/360`，回放目标从本机自动化目标/活动校准解析目标
  `counts/360` 并自动换算；不得用工作流倍率参数近似补偿。
- 可下载工作流携带精确 NodeRef、Workflow Source 和内容寻址 Blob；本机 target、credential 与校准
  不进入可移植资源，导入后必须经过兼容性预检和本机重绑定。

## Terms

- **Authoring Projection:** 从 Node Contract 与 Data Type 派生、可严格重开的编辑事实。
- **Macro:** 可逐动作编辑的简易输入自动化资源。
- **InputClip:** 保留原始时序、轨迹和交叠输入的精准录制资源。
- **Stage:** 多个相邻交付项组成的一次产品闭环和集中验收边界。
