---
title: 定义 Capability 与 Target Planning
label: wayfinder:grilling
parent: ../map.md
status: closed
assignee:
blocked_by:
  - define-node-contract-metaschema.md
---

# 定义 Capability 与 Target Planning

## Question

host OS、architecture、automation target、host application、外部 effect、credential 与用户授权应如何分别建模，并由 Compiler 形成可执行 capability plan，使节点只能获得被 Program 声明且被 Run policy 授权的能力，Script 和第三方节点也不能形成隐式能力升级？

## Resolution

已接受。Yotta 采用“声明、计划、绑定、授权、调用”五段式 capability 模型；Program 声明不能自我授权，Run Grant 也不能授权 Program 未声明的能力。WASI 的无 ambient authority 原则与 MCP 对 audience/resource binding、least privilege、禁止 token passthrough/confused deputy 的要求作为跨 builtin/Wasm/Process/MCP 的共同安全基线：[WASI](https://wasi.dev/)、[WIT resources](https://component-model.bytecodealliance.org/design/wit.html)、[MCP Authorization](https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization)、[MCP Security Best Practices](https://modelcontextprotocol.io/docs/tutorials/security/security_best_practices)。

### 1. 六类事实必须分离

1. **Host Profile** 是宿主探测并签名/信任的平台事实：OS、architecture、Yotta host API generation 与 installed provider inventory。Source、节点 config 和插件 manifest 不能伪造它。
2. **Automation Target** 是 provider 可操作的具名实例，例如 host desktop、Android device、browser session 或 After Effects instance；Target identity 不等于 capability，也不包含 credential。
3. **Capability Definition** 使用绝对 `/vN` ID 与 semantic digest，定义封闭 operation set、允许的 target kind、scope schema、是否需要 credential、风险/consent class 和 provider ABI。改变 operation/scope/authorization 语义必须发布新版本。
4. **Capability Requirement** 属于 Effective Node Contract，使用局部 requirement ID 引用精确 Capability Definition，并声明最小 operation 子集、target slot、credential slot 与通过 scope schema 验证的 canonical scope。它不携带 target instance、secret 或 grant。
5. **Capability Plan** 是 Compiler 写入 Program 的、按 node/requirement 排序的 attributed manifest；每项保留来源 graph/node、精确 capability ref、operations、target/credential slot 与 scope。Plan digest 进入 Program identity，UI/MCP 由它生成 permission delta。
6. **Run Grant / Binding** 是宿主 Policy 在 Run 开始前产生的运行事实：把 plan slot 绑定到 provider、target 与 credential metadata，并授权精确 operation/scope。Credential material 始终留在 OS credential store/provider 内。

### 2. Compile 与 planning

- Compiler 只根据 trusted Catalog + Effective Node Contract 生成计划，不读取当前机器、credential store、用户 approval 或 mutable provider registry；相同 Source/Catalog/build 必须产生相同 plan。
- 每个 requirement 必须保留来源节点，不能只 union 成字符串集合；同一 capability 的不同 scope/target slot 也不能合并，以便准确显示权限 delta、错误与审计 lineage。
- `requestedCapabilities` 不再是自由字符串 allowlist；最终 Source authoring projection由精确 requirement delta 派生。旧字段在 3.1 migration 中删除，不作为第二授权事实。
- Target Planner 在 Run admission 时使用 Host Profile 和 target inventory 为每个 slot 选择/验证 binding。零候选、多个候选未消歧、provider ABI/digest 不匹配均在执行任何节点前失败。

### 3. Grant 不变量

- grant 至少绑定 `programHash + capabilityPlanDigest + runId + principal + providerId + targetId + operations + canonical scope + issuedAt/expiry + policy generation`；必须短期、可撤销、不可猜测且不能跨 Run/Program/target/provider 复用。
- Policy 只能缩小 plan，不能增加 operation、扩大 scope 或替换 target kind。权限扩大必须重新 planning/consent；错误重试不能自动 step-up。
- Program/Run artifact 只持久化 grant digest、policy generation、non-secret binding metadata 和 consent lineage；opaque token、OAuth token、API key、cookie 与 refresh token不得写入日志、trace、ValueEnvelope 或插件协议。
- headless、GUI、AI、MCP、builtin、Wasm 与 Process Node 使用同一 admission API；不存在“本地可信所以跳过 grant”的分支。

### 4. 执行边界

- Interpreter 只把单个 invocation 所需的 narrowed Capability Session 交给 adapter；adapter 不能取得全局 `ServiceBundle`、credential store、target registry 或其他节点的 grant。
- 每次调用重新核对 grant 的 Program/Run/node/requirement/operation/scope/target/provider binding、expiry 与 revocation；只按 capability ID 或 entrypoint 查服务不足以授权。
- Script 与第三方节点只能调用显式 import/protocol operation；动态字符串、prompt、tool result、插件返回值和 Resource Reference 都不能新增 capability。
- host/infrastructure authorization error 使用稳定命名空间并区分 unsupported host、target unavailable、credential unavailable、consent required、denied、expired/revoked 和 quota exceeded；不得伪装为节点业务 error channel。

### 5. 平台与 target

- platform compatibility 是 `Host Profile × provider descriptor × target descriptor` 的 planning 判断，不进入 Data Type，也不从 capability 名称猜测。
- Windows stable 必须有真实 provider smoke；Linux/macOS 可在 provider 不可用时 strict compile 并在 planning 返回 structured unsupported，不得悄悄选择另一实现。
- remote target 的 transport/authentication 是 provider 内部能力；Program 只保存 target slot/constraint，不能保存活 session、socket、HWND、process ID 或 bearer token。

### 6. Conformance

- 覆盖 Program 声明但无 grant、grant 有但 Program 未声明、operation/scope 扩大、错误 audience/target/provider、跨 Run replay、expiry/revoke、credential 缺失和 confused-deputy token passthrough。
- 同一 Capability Plan 必须为 GUI、headless、MCP 和插件 host 生成相同 permission delta；拒绝路径在任何 adapter 代码运行前发生。
- capability/ref/operation/scope/plan 的 Go/TS/WIT/Protobuf projection 使用同一 golden vectors，不允许各 host 自建 scope matching。

## Implementation status

Capability Definition、Requirement 与 sealed attributed Plan 已进入 `internal/capability`；Node Contract 使用 exact requirements，Catalog 绑定 definitions，Compiler 把 plan artifact 写入 Program identity。Source 的自由 `requestedCapabilities` 与 preview string-grant 参数已删除。Run Grant、Host Profile/Target Planner 和真实 admission 仍由 Program/Run 纵向切片实现；不得为此恢复字符串 allowlist。
