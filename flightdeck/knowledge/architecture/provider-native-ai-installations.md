---
kind: note
summary: "Yotta 3.1 AI 必须以内容寻址 Model Profile、PromptManifest 与 ToolSet 组成可信安装边界；provider/target/credential/consent 由启动时 Host Profile 冻结，运行时数据不得升格为高权限指令。"
activation: action
read_when: "新增或修改 AI provider、模型设置、PromptManifest、ToolSet、AI 节点、workflow consent、credential binding、Host Profile、Policy/Run Grant 或 AI trace 时"
recheck_when: "Model Profile/PromptManifest/ToolSet digest、provider ABI、供应商原生 API、AI capability scope、credential store 或 workflow consent preimage 改变时"
---
# Provider-native AI installation contract

Yotta 3.1 不把 AI 当成可随调用拼接的 endpoint。设置只声明模型安装档案：稳定 slot、供应商原生协议、exact model、输出预算、已验证能力、固定 token pricing 和评估状态。档案被 canonical seal 后，启动装配为以下锁定关系：

- 相同 Model Profile 可共享一个 native provider 实例，但每个 slot 必须有独立 target 与 credential binding。
- OpenAI adapter 只走 Responses API，Anthropic adapter 只走 Messages API。不得通过 Chat、prompt JSON、模型列表或 endpoint 猜测做兼容回退。
- API key 只按 slot 存入 OS credential store；settings、RPC 返回、日志和 trace 不得含明文 secret。
- workflow consent 是 slot、profile digest、provider ABI 和允许 operation 的内容摘要。档案语义变化后旧 consent 必须失效，不得自动迁移或扩大。
- Host Profile 在应用启动时冻结 provider artifact、target、credential binding 与 consent。Policy 只能对 exact proposal seal bounded Run Grant；运行中不能热插入 ambient provider。
- provider 结果必须保留 requested/resolved model、finish、供应商 request identity、token usage 与按 profile pricing 估算的 cost；未知响应类型、缺失 usage 或能力不匹配要 fail closed。

Trusted instruction boundary 也必须是安装时/构建时冻结的 artifact：

- PromptManifest 与 ToolSet 使用独立 versioned hash domain、canonical bytes、严格 reopen、unknown-field rejection 和显式 byte/depth/count budget。
- provider 的 system/developer 高权限字段只能来自 strict-opened PromptManifest；workflow config、用户输入、网页、OCR、检索内容和 tool result 只能进入 typed untrusted blocks。
- Workflow Source 与 Program 只引用 prompt/profile/toolset identity，不持久化任意 system/developer 文本；旧 instructions override 不得保留兼容 fallback。
- 内置 prompt digest 必须进入 implementation lock。AdapterAction 只记录 prompt/schema/toolset digest 与脱敏 model/provider/usage/cost facts，不记录原始 prompt、schema、trusted instructions、tool result、provider transcript 或 secret。
- package-owned PromptManifest 只有在 Node Package trust 明确验证 package identity、签名与 owner namespace 后才可成为 trusted source；仅有合法 JSON 或自报 owner 不构成信任。

Agent continuation 属于 provider-owned opaque state，不能成为 workflow 数据或 durable transcript：

- OpenAI Responses 在 provider storage enabled 时用 previous_response_id 续接，但每轮仍重发 exact trusted instructions、ToolSet 与 parallel policy；store:false 时必须按顺序回放完整 native output items，并请求/保留 encrypted reasoning content。
- Anthropic Messages 必须原样保留并回传 assistant content blocks，特别是 thinking/signature；所有 client tool_result 集中在紧随其后的单条 user message，且每个 pending tool_use 必须精确匹配。
- continuation state 采用 copy-on-write；响应、schema 或 pending-call 校验失败时不得推进旧 state。opaque state 有独立 byte budget，session 完成或关闭后必须清除。
- Agent capability scope 只能开放 agent-start/agent-continue，不能混入普通 Generate；tool authority 必须来自 exact ToolSet 与宿主 binding/approval，模型输出永远不能授予 capability。
- Agent run 必须同时约束 input/output token、estimated cost、wall time、iteration、tool-call count 与 parallelism；缺失 usage/cost 或任何计数溢出都按 budget exhaustion fail closed。

设置测试只能对 exact profile 发起一次 provider-native generation。成功发现 endpoint 或列出模型不构成可运行性证明。
