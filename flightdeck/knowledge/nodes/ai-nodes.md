# AI nodes and provider-native runtime

当前单一路径是 `internal/nodes` contracts → `internal/noderuntime` adapters → `internal/ai` provider。旧 `internal/nodes31`/`nodes31runtime` 名称不存在，也不得恢复产品版本后缀 package。

- Generate/Extract/Agent 都固定 exact NodeRef、typed ports、config schema、prompt/tool artifact 和 `ai-generation` capability requirement。
- Workflow 只选 installation slot并提供低信任 typed input；endpoint、provider、model、API key 和高权限 instruction 不能由运行时 prompt 猜测。
- OpenAI 使用 Responses API，Anthropic 使用 Messages API；不做通用 Chat/fence/endpoint 自动兼容回退。
- API key 只存在 OS credential store；Settings/RPC/trace 不回传 secret。
- Provider outcome 保留 provider/model/request identity、finish、usage/cost 与 typed output；未知或不完整响应 fail closed。
- Agent continuation 是 provider-owned bounded opaque state，不成为普通 workflow 数据或 transcript。

ModelProfile、PromptManifest、ToolSet、provider ABI、credential binding 或 consent scope 改变时生成新 installation digest 并撤销旧 consent。配置通过原子 generation 发布；正在运行的 Run 持有旧代，新 Run 使用新代，不依赖应用重启。

AI authoring 只能提交 typed PreparedPatch，由用户审查后经 Application revision CAS 保存；模型不能直接写文件、安装 provider、授予 capability 或执行旁路 Run。
