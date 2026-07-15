---
kind: note
summary: "Yotta 3.1 AI 必须以内容寻址 Model Profile 安装到稳定 slot；provider/target/credential/consent 由启动时 Host Profile 冻结，运行时不得发现模型、替换别名或降级协议。"
activation: action
read_when: "新增或修改 AI provider、模型设置、AI 节点、workflow consent、credential binding、Host Profile、Policy/Run Grant 或 AI trace 时"
recheck_when: "Model Profile digest、provider ABI、供应商原生 API、AI capability scope、credential store 或 workflow consent preimage 改变时"
---
# Provider-native AI installation contract

Yotta 3.1 不把 AI 当成可随调用拼接的 endpoint。设置只声明模型安装档案：稳定 slot、供应商原生协议、exact model、输出预算、已验证能力和评估状态。档案被 canonical seal 后，启动装配为以下锁定关系：

- 相同 Model Profile 可共享一个 native provider 实例，但每个 slot 必须有独立 target 与 credential binding。
- OpenAI adapter 只走 Responses API，Anthropic adapter 只走 Messages API。不得通过 Chat、prompt JSON、模型列表或 endpoint 猜测做兼容回退。
- API key 只按 slot 存入 OS credential store；settings、RPC 返回、日志和 trace 不得含明文 secret。
- workflow consent 是 slot、profile digest、provider ABI 和允许 operation 的内容摘要。档案语义变化后旧 consent 必须失效，不得自动迁移或扩大。
- Host Profile 在应用启动时冻结 provider artifact、target、credential binding 与 consent。Policy 只能对 exact proposal seal bounded Run Grant；运行中不能热插入 ambient provider。
- provider 结果必须保留 requested/resolved model、finish、供应商 request identity 和 usage 的脱敏事实；未知响应类型或能力不匹配要 fail closed。

设置测试只能对 exact profile 发起一次 provider-native generation。成功发现 endpoint 或列出模型不构成可运行性证明。
