---
kind: note
summary: "3.1 校验按 Source/schema、Compiler、Admission、Executor/adapter 分层，但语义均来自 sealed Contract/Catalog；不能在前端或 runtime 建第二套规则。"
activation: action
read_when: "新增配置校验、类型/端口诊断、target/capability 检查或 runtime contract check 时"
recheck_when: "ConfigValidator、Compiler diagnostic、Admission、Projection 或 adapter validation 改变后"
---
# 3.1 validation layers

- **Parse/schema**：严格格式、unknown field、budget、config schema 和稳定引用。
- **Compiler**：exact NodeRef、端口/channel、类型、state、GraphCall、instruction、capability plan 和 implementation lock。
- **Admission**：host feature、provider/target/credential candidate、policy/consent 和 durable QUEUED Run。
- **Executor/adapter**：Program/Grant/implementation 复验、runtime value reseal、真实宿主请求与 action journal。

这些层负责不同时间点，但 machine semantics 必须来自同一 sealed Data Type/Node Contract/Catalog。前端可展示 Projection 和预检结果，不能发明独立 assignability、required、capability 或 error-route 规则。

新增规则时先决定它能否仅依赖 Source/Contract；能则放 Compiler/ConfigValidator并生成稳定 diagnostic。需要已安装目标时放 Admission；需要真实宿主状态时放 adapter。不要用 runtime fallback 修复编译期错误，也不要把 target unavailable 报成数据类型错误。
