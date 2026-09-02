# AI Run 诊断与工作流修复 context

## Scope

- Run 结束后，时间线应能解释节点实际发生了什么，并支持从事件定位回 Workflow 编辑器中的 Graph/Node。
- 模板匹配超时需要保留足以调优的有界证据，例如最高分、采用阈值、候选位置、模板来源与目标分辨率。
- Workflow 界面提供 AI 诊断入口；AI 能读取 Run 证据、节点契约与帮助，并提出或执行可审查的 Source 修改。
- AI 修改后编辑器必须能检测 revision 变化并安全重新加载，不能静默覆盖未保存编辑。
- 本机 MCP 是 AI 工具的候选 seam，但现有能力、身份与修改语义必须以生产代码为准。

## Constraints

- 模板匹配逐帧热路径不得增加新的权限包装、重复配置解析或 map/mutex 查询；运行证据在已有匹配计算结果上
  有界聚合，并在状态/事件边界写入。
- 用户阈值与派生模板策略是不同概念。默认用户阈值暂时保持 0.85；任何派生补偿都必须通过误匹配数据验证，
  不能因为单个样本得分 0.824802 就全局降至 0.75。
- AI 对 Workflow 的修改作用于可编辑 Workflow Source，遵守 revision conflict 与 schema/compiler validation；
  不直接改 Program、Run 历史、生成 bindings 或本地用户数据文件。
- 运行证据和日志不得泄漏凭据、用户输入、完整敏感截图或不受限 Blob；AI 读取采用有界、脱敏的结构化投影。
- 面向用户、Timeline、MCP 或 AI 的错误不得依赖 Go error string、包名、adapter 实现名或英文子串；稳定错误 ID
  与 typed params 是翻译和自动诊断的唯一契约，内部 cause 只进入受控日志/支持证据。
- 产品尚无外部兼容负担，本阶段允许破坏旧错误信封、Node Contract 和 Run Evidence 格式；不维护字符串错误兼容层。

## Terms

- **Run Evidence**：某次 Run 已发生事实的有界结构化记录，可由 Timeline 事件、值引用与受控附件组成。
- **Diagnostic**：基于 Source、契约或 Run Evidence 得出的可操作问题说明；它不是 Run 事实本身。
- **Log**：面向维护者的进程/模块运行记录，用于解释系统实现行为；不承担产品级 Run 历史或编辑器导航。
- **Workflow Repair**：基于精确 Source revision 提出的结构化修改，经验证和用户审查后形成新 revision。
- **Error ID**：跨版本稳定、可翻译的机器标识；其参数由注册的 typed schema 约束，不包含最终展示文本。
- **Node Outcome Evidence**：节点一次 attempt 对正常结果、未满足条件或失败所保留的有界结构化事实。

## Delivered decisions

- 用户阈值保持单一默认 0.85；不增加全局“自适应阈值 0.75”。派生模板的实际最高分与阈值进入 Run Evidence。
- Timeline 使用 Workflow ID、Source hash/revision、graph path 和 node ID 定位；AI 只在 exact revision CAS 成功后
  重新加载，脏编辑不会被静默覆盖。
- MCP 与内置 AI authoring 共用结构化 Run Evidence 和 typed patch，不建立平行工作流修改协议。
- Timeline 保存有界数值证据；显式诊断工具按需保存截图，不在每次轮询中持久化画面。
