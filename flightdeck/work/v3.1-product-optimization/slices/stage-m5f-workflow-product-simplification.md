# M5f — 工作流产品模型简化

## Goal

让用户只面对一个“工作流”概念：本地创建和外部导入的工作流出现在同一列表、使用同一运行入口；
Release 只保留为内部版本来源与回滚快照。删除工作流、应用、目标、HTTP 与 AI 配置上的重复 consent，
把“用户已配置/选择本机能力”作为可使用该能力的唯一产品动作。

## Product contract

- 工作流列表只有一个，不再上下分成“已安装工作流”和“工作流源”。
- 本地工作流标记为“可编辑”；外部导入工作流标记为“外部导入 · 只读”。两者都可直接运行、配置计划和
  查看运行结果；只读工作流可显式创建可编辑副本。
- Release、Installation 和 digest 是内部持久化、更新、来源与回滚事实，不作为一级产品对象或运行许可。
  用户界面统一称“工作流”“来源版本”“更新”“回退”。
- 配置应用、自动化目标、HTTP 连接或 AI 账号后即可以被工作流使用，不再提供授权、撤销授权、全部授权按钮。
- 手动运行不需要首次授权，更新后也不需要重新授权。
- 计划不保存或检查 Release consent。启用与触发只检查 lifecycle、依赖、目标和凭据等真实可执行条件。
- 更新保留兼容的本机目标和凭据。capability diff 继续用于更新说明，但不暂停计划、不清除许可；若新增依赖、
  目标或凭据确实缺失，Readiness 自然阻止运行并给出配置动作。
- Compiler capability plan、admission、Run Grant、Resource Broker 和 arm 边界继续在后台约束实际调用，
  不新增用户确认。

## Implementation guide

### 1. 删除重复 consent

1. 从 Workflow Installation `Configuration`、Readiness blocker/action、设置服务和 Wails transport 删除
   `RunConsentRelease`、`ScheduleConsentRelease` 与 `GrantInstallationConsent`。
2. `EvaluateReadiness` 只投影 lifecycle、dependency、target 和 credential；手动与计划使用相同 readiness。
3. 更新/回退不再清空 consent，也不因 capability scope 变化暂停计划。
4. 从 AI、HTTP、Application、Automation 的持久设置、安装 draft、manifest projection、服务方法和设置 UI
   删除 `WorkflowConsent`。配置有效且可安装即为 available；系统/设备自身的 UAC、OAuth、ADB 授权不伪装成
   Yotta 二次授权。
5. Content Catalog 新 migration 删除两个 legacy consent 列；settings reader 忽略并在下次写入移除旧字段。

### 2. 合并工作流产品列表

1. Workflow 模块提供统一 library projection，组合 editable Source 与 imported Installation，统一完成搜索、
   排序、分页和总数计算。
2. 每一项显式返回 `kind`、`readOnly`、可选 `installationId`、来源版本和 readiness；前端不推断内部对象。
3. 删除工作流页顶部 Installation 面板。统一行根据 projection 显示：
   - editable：编辑、运行、导出、元数据与删除；
   - imported/read-only：只读标签、运行、目标与凭据设置、更新/回退、创建可编辑副本。
4. 计划编辑器继续引用稳定的内部 `installationId`，但用户文案和选择器只显示工作流名称与来源标签。
5. Release/Installation 专有词只保留在诊断、存储和开发接口中；普通 UI/i18n 改为工作流和版本语言。

## Data compatibility

- 保留现有 `workflow_releases`、`workflow_installations` 和配置记录，避免破坏已导入内容、目标/凭据和回滚历史。
- migration 只删除无价值的 consent 摘要，不删除 Source、Release、Installation、schedule 或 binding。
- 旧 settings JSON 中的 `workflowConsent` 允许 reader 接受但不再进入当前模型，下一次保存自然清理。
- 现有按 Installation ID 保存的 schedule 不改身份，仅删除 consent gate 与相关重新授权/暂停行为。

## Acceptance scenarios

1. 新建本地工作流与外部导入工作流同时出现在一个搜索结果表中，没有第二个“已安装”区域。
2. 外部导入项明确只读；可直接运行，也可创建可编辑副本，副本与原项同时存在且互不覆盖。
3. 新配置应用或自动化目标后，工作流无需再点击授权即可运行。
4. 首次手动运行、普通内容更新和回退均不出现授权按钮或 consent blocker。
5. 已启用计划在仅 graph/resource/capability 描述变化的更新后保持启用；缺少新增 target/credential 时，
   readiness 只报告真实缺项。
6. 旧 Content schema 7 profile 升级后工作流、binding、schedule、previous version 均保留，
   legacy consent 列消失。
7. 定向 Go/前端测试、`task check` 和 Windows WebView 工作流列表/运行/计划旅程通过。

## Current

完成。产品只暴露一个工作流列表；外部导入项以“外部导入 · 只读”标记并在同一行提供运行、设置、
更新/回退和编辑副本。Release 只保留为内部版本、来源与回滚事实，普通界面统一使用“版本”。
AI、HTTP、桌面应用、自动化目标以及工作流运行/计划的二次 consent 已移除，配置完成即可使用。

## Next

M5f 已闭环；后续按主计划评估 M6，不再为本阶段保留实现动作。

## Verification

- Content Catalog schema 8 单向删除 legacy workflow consent；旧 settings 字段由兼容 reader 接受并在
  下次保存时清理，不删除工作流、binding、schedule 或版本历史。
- `task check` 退出 0：37 个受影响 Go 包、Wails 16 services / 149 methods / 235 models、
  前端 84 个测试文件 / 357 项测试全部通过。
- Windows WebView smoke `20260726-165705` 退出 0，统一列表、导入工作流设置、更新、回退、编辑副本、
  编辑器、资源与计划旅程通过；关键截图已人工检查。
