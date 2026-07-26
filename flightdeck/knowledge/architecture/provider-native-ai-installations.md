# Provider-native AI installation contract

Yotta 不把 AI 当成可随调用拼接的 endpoint。设置只声明模型安装档案：稳定 slot、供应商原生协议、exact endpoint/model、输出预算、已验证能力、固定 token pricing 和评估状态。档案被 canonical seal 后，通过 installation generation 发布以下锁定关系：

- 相同 Model Profile 可共享一个 native provider 实例，但每个 slot 必须有独立 target 与 credential binding。
- OpenAI adapter 只走 Responses API，Anthropic adapter 只走 Messages API。不得通过 Chat、prompt JSON、模型列表或 endpoint 猜测做兼容回退。
- API key 只按 slot 存入 OS credential store；settings、RPC 返回、日志和 trace 不得含明文 secret。
- workflow consent 是 slot、profile digest、provider ABI 和允许 operation 的内容摘要。档案语义变化后旧 consent 必须失效，不得自动迁移或扩大。
- 同一 sealed generation 原子投影 provider artifact、target、credential binding、Host Profile 与 consent；正在运行的 Run 持有其 generation lease，新 Run 只能看到已发布的新代。配置变化不得靠 ambient provider 热插入，也不应要求正常重启应用。
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

Offline evaluation 与 upgrade gate 是安装的一部分，不是 UI badge：

- mandatory EvalSuite 必须固定 corpus、deterministic grader version、baseline 与 pass/safety/token/cost/latency thresholds，并以 strict canonical artifact reopen。
- EvalReport 必须由 suite 对每个 case 精确匹配 observation 后导出；decision 与 aggregate metrics 必须可从 case results 和 thresholds 重算，unknown field、结构超限、重复 case 或 report drift 都 fail closed。
- evaluation subject 只覆盖 model runtime identity；upgrade candidate 另将 subject 与当前 Generate/Extract/Agent/Authoring PromptManifest、Agent/Authoring ToolSet、三个 AI Node Contract semantic digest 排序绑定。任一 prompt/tool/schema/code upgrade 都使旧 candidate stale。
- suite digest 与 exact report digest 都进入 ModelProfile；因此重新评估、report replacement、approved/rejected 变化都会改变 profile digest 并撤销旧 workflow consent。
- Settings 可保留 unverified、rejected 或 stale profile 以便测试/重新评估，但只有 approved report 且 exact current candidate 才能进入 Host Profile。semantic profile edit 自动降级为 unverified 并清除 report、suite 与 consent。
- canonical report 通过 ApplyEvaluation 显式导入、RevokeEvaluation 显式撤销；GrantWorkflowUse 必须再次验证 exact current candidate。task check 必须 regrade tracked corpus 并拒绝 report drift。

AI authoring 仍是 typed Application client，不获得文件或 admission authority：

- Authoring ToolSet 只开放 catalog search/describe、workflow inspect、pure patch proposal、compile/preview 与 diagnostic explain；模型输出只能形成 opaque PreparedPatch，不能直接保存 Source。
- review artifact 必须绑定 base/new revision 与 exact source hash，并列出 normalized changes、diagnostics、capability/credential/target delta、risk、usage 和 redacted provenance；敏感 input 只保留 trust class/digest/size。
- accept 必须重新验证 base revision/hash，并提交已审查的 exact candidate；revision conflict、reject 或 session close 均丢弃 proposal。权限扩大必须有额外显式确认，不能由模型或旧 approval 推导。

设置测试只能对 exact profile 发起一次 provider-native generation。成功发现 endpoint 或列出模型不构成可运行性证明。
