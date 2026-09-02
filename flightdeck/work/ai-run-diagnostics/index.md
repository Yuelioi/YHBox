# AI Run 诊断与工作流修复

## Goal

让用户和 AI 能从一次 Run 的异常或低质量结果定位到具体 Workflow/Graph/Node，读取足够的运行证据与节点帮助，
提出或执行可审查的 Workflow Source 修改，并在编辑器重新加载后验证结果；同时明确 Timeline、Diagnostic 与 Log
各自承担的职责。

## Status

Finished

## Current

已完成运行证据职责核验：Timeline 应继续作为 Run Evidence 产品视图，Diagnostic 是基于证据的解释，Log 只承担
维护实现上下文。已确认模板 score 在轮询后丢失、Run view 缺 Source revision、本机 MCP 已有完整 typed Workflow
authoring 基础但没有 Run Evidence 工具、编辑器已有 node focus 和 revision conflict/reload seam。交付可复用这些
深 module，不需要另建平行 AI 工作流编辑协议。

范围已扩展为全节点结果与错误契约治理：需要审查每个内置 Node Type 的 timeout/not-found/exhausted/failed
语义，建立稳定 error ID + typed params + internal cause 分层，消除原始 adapter/package/Go 错误和英文字符串分类，
并让翻译、Timeline、MCP 与 AI 共用同一机器契约。

Stage 0 已开始实施：Wails envelope 改为 `id + params`，未分类错误统一为 `system.unexpected` 且不再返回 raw
cause；Admission/Workflow patch provider、revision conflict 和前端 transport 已迁移，旧 code/message/details
字符串兼容按产品现状直接移除。首轮仓库增量门禁通过。Automation Target health 也已改为结构化 ID/params，
移除 adapter cause 和英文 Message；正式更新 Wails RPC 契约后，第二轮仓库增量门禁也已通过。

Run 诊断主链已贯通：新增 canonical/bounded Problem Params module，Node Contract v2 可声明 typed error params，
Scheduler 校验并把 params 写入 Journal/terminal failure；Run 保存 Workflow ID、Source hash/revision。Template timeout
聚合最佳分/阈值/候选框，Observation、Window 与 Retry timeout/exhausted 发布统一 status evidence。Timeline 展示
来源 revision、模板证据和显式“定位节点”，并可直接打开 AI 诊断。MCP 新增 `run_list`/`run_get`，所有 MCP
tool error 也改为结构化 Problem。AI proposal 可锁定 Run 并通过只读 `run_get` 诊断后生成 typed patch；设置新增
诊断 AI role。用户可见 async result/error string 已迁移为 Problem；未分类 cause 仅通过 operation ID/error type
关联维护日志。跨契约增量门禁通过，CLI WebView smoke 修复长期过期 fixture/selector 后通过并保存全套截图。

## Next

None.

## References

- [稳定上下文](context.md) — 本阶段产品范围、约束与已确定术语。
- [阶段计划](plan.md) — Wayfinding 决策与后续交付顺序。
- [错误契约源码审计](references/error-contract-audit.md) — 当前泄漏路径、timeout 差异、统一模型与验收标准。
- [错误契约开发指南](../../knowledge/errors/error-contract.md) — 后续 Work 修改 Problem、RPC、异步失败和持久证据时的统一规则。

## Progress

- 2026-09-02 核验 Run Ledger、模板节点、Timeline UI、日志和 MCP authoring 生产代码；确定职责边界并形成七阶段
  交付骨架。默认用户阈值保持 0.85，派生模板策略延后到有误匹配/漏匹配样本的独立研究阶段。
- 2026-09-02 完成跨 Node/runtime/adapter/service/frontend/MCP 错误契约审计；确认已有 apperr/code/params 基础
  未贯通，Run 丢失 failure params，多个 async result/health/MCP 路径泄漏原始字符串，timeout evidence 不一致。
- 2026-09-02 按允许破坏更新的决定开始 Stage 0：删除 Wails raw message/details 信封与前端英文 message fallback，
  引入 `id + params` 投影和 `system.unexpected` 安全 fallback；RFC 9457/OTel 用于校准产品 Problem 与内部 cause 分离。
- 2026-09-02 首轮 `task check` 通过（38 个 Go package、bindings、491 项前端测试）；随后迁移 Automation Target
  health 为 `id + params`，并新增 adapter cause 不泄漏回归。
- 2026-09-03 正式更新 Wails RPC 快照并完成第二轮 `task check`：38 个 Go package、bindings、i18n、lint、
  typecheck 和 491 项前端测试通过。
- 2026-09-03 推进 Run/AI 主链：Node Contract 升至 v2，新增 typed Problem Params、Run Source attribution、三类
  wait evidence、Retry exhausted evidence、Timeline 定位/AI 诊断、Run MCP、结构化 MCP problem 和诊断 AI role。
- 2026-09-03 修复隔离 WebView smoke 的空 Source fixture及长期过期的管理页/画布 selector；CLI smoke 通过，
  生成 Workflow、Run 状态、AI review、资源、计划和设置截图。真实 2K 模板证据沿用先前 WGC 成功采样。
- 2026-09-03 完成封板：用户可见同步/异步错误统一为稳定 Problem，Run/Timeline/MCP/AI Repair 全链路交付；
  `task check`、CLI WebView smoke、Node/Catalog/version compatibility 与 AI eval 全部通过。
- 2026-09-03 正式 `task build` 通过：生成 `bin/Yotta.exe` 及 worker/CLI/capture DLL，Windows 版本元数据和隔离
  desktop startup smoke 通过。
- 2026-09-03 完成 AI 提案入口与模型资格 UX 收尾：撤销/重做/搜索移至编辑器左侧上下文区，AI 提案提升为
  “工具”前的独立主面板入口；诊断角色与提案模型仅显示已批准且支持 Tool Calling 的候选，无候选时改为明确
  原因与设置入口，不再展示无法操作的空下拉。CLI WebView smoke、`task check`（495 项前端测试）和正式
  `task build` 全部通过。
- 2026-09-03 根据产品反馈移除评估状态的诊断准入门槛：诊断 AI 和 AI 提案现在只要求模型声明 Tool Calling，
  `unverified`/`rejected` 继续作为质量信息展示但不阻断配置或执行；前后端回归、`task check`（495 项）及正式
  `task build` 通过。
- 2026-09-03 参考 YomiFX 的 Project Thread 模型，将 AI 提案升级为 Workflow-scoped 侧栏对话：支持持久化
  会话历史、新建/切换会话、连续上下文、普通问答、实时工具进度和消息内候选审查；每个 Workflow 的历史物理
  隔离保存于 `data/ai-conversations`。修复 Windows Codex app-server 控制台弹窗，并将 provider failure 投影为
  可翻译 Problem。真实 Codex 动态工具 smoke、CLI WebView 侧栏截图、AI eval 8/8、`task check`（496 项）通过。
- 2026-09-03 针对真实诊断请求仍显示未知错误复盘现场：失败发生在消息落盘前且旧版没有保留 cause。补齐模型
  profile/provider 前置错误 ID、前端/Wails transport 分阶段 fallback，以及“首步即落盘”的失败证据；用本机实际
  `model-3 / gpt-5.6-sol` 配置完成原生连接 smoke，并再次通过 `task check`（496 项）和强制生产构建。
- 2026-09-03 将完整错误模型沉淀为 Knowledge：明确 canonical envelope、逐阶段 projection ownership、
  Run/AI/异步 durable evidence、frontend unknown policy、ID/params 规则与必需验证矩阵，并从 Wails/UI 指南链接。
- 2026-09-03 完成真实端到端 AI Run 诊断：用本机 `model-3 / gpt-5.6-sol`、工作流“异环 看电影”和真实
  cancelled Run 成功读取 evidence、生成候选修订 41、编译并预览权限。修复中文标题按 UTF-8 bytes 误判超长，
  并兼容模型实际生成的 `commands` envelope 与扁平 `set-node-binding`、`inputId/portId` 表达；新增
  `cmd/ai-authoring-smoke` 作为可重复真实验收入口。
- 2026-09-03 根据 owner 反馈撤销向模型注入完整 Authoring Patch schema 的方向：阈值等数值修复改用
  `workflow_set_numeric_input(graphId,nodeId,inputId,value)` 窄动态工具，host 内部准备 typed candidate；本机 MCP
  同步新增 `workflow_set_input_value`，外部 AI 可用相同节点/输入语义原子修改，不必学习完整 Command union。
