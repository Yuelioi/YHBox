---
slice: "09"
title: 工作区一致性与黄金路径
status: pending
---

## Outcome / Question

把前八个 Slice 已恢复的工作流、目标、资源、录制、调试与计划能力串成发布前连续旅程；修正只在跨页面、重开或真实目标下出现的割裂，不再新增平行模型。

## Completion criterion

- 从新建工作流开始，可连续完成默认目标选择、宏/精准录制/视觉模板创建、资源绑定、编译、运行、调试、保存重开与计划引用。
- 工作流页、资源库、计划页和编辑器在标题、筛选、分页、空状态、选择、错误反馈与主操作层级上遵循同一产品语言。
- Windows UAC 真机目标与至少一个无需桌面目标的 portable 工作流通过；Android 入口与平台适用性不被 Windows 默认体验遮蔽。
- 数据目录不再向用户暴露测试版本语义；升级前知识和兼容说明在 3.1 发布收口时归档或清理。
- 失败点必须定位到 Source、Installation/Target、Asset、Compiler、Admission、Runtime 或 UI projection 的唯一责任层，禁止页面临时兜底制造第二事实源。

## Verification

- 使用独立验收数据创建而不是复用手工修补 fixture，保存后关闭并重开验证 durable truth。
- 执行 `task check`、production `task build`、`task webview:smoke:full` 与 UAC 真机旅程；检查关键截图、日志和 Run 时间线。
- 只在 Slice 完成后进行整阶段批量验收并提交。

## Out of scope

- 新增 OCR/AI 算法、第三方节点 UI ABI 或新的执行 runtime。
- 为未发布 3.1 保留已确认无价值的旧数据兼容分支。
