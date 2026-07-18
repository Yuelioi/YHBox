---
kind: checklist
summary: "3.1 Node Contract 风格：稳定 URI/SemVer、lowercase port ID、严格 schema、分离 data/exec/error/status、完整 execution/capability 语义。"
activation: action
read_when: "编写或修改 internal/nodes 中的 Data Type、Capability Definition、Node Contract、端口、config schema 或 authoring metadata 前"
recheck_when: "nodecontract Draft/normalization、port ID 规则、ExecutionSpec、Capability Requirement 或 Authoring Projection 改变后"
---
# 3.1 Node Contract style

## 身份

- `nodeTypeId` 使用稳定 lowercase URI，如 `https://schemas.yotta.dev/nodes/automation/press-keys`；不含 `3.1`、`v31` 或实现包名。
- `version` 是独立 SemVer；破坏 machine semantics 时升级，展示文案变化不改 semantic digest。
- TypeRef/CapabilityRef/validator/effect ID 使用各自版本化 URI 与精确 semantic digest。

## 配置与端口

- config schema 使用 Draft 2020-12、固定 `$id`、`additionalProperties:false`、明确预算和默认值。
- config key 与 port ID 使用 `^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$`；同一概念复用已有词汇，例如 `in`、`completed`、`failed`、`timeout`、`point`、`region`、`slot`。
- data inputs/outputs、exec inputs/outputs、error outputs 分开声明。空频道保持空，UI/runtime 不猜通用 `out`。
- required/default 都是 machine semantic；默认 JSON 必须能由 exact TypeRef reopen，不用字符串冒充数字/对象。
- runtime carrier 必须显式 ResourceLeaseBinding；path、handle、channel 或 pointer 不作为普通 data value。

## 执行与失败

- 填完整 class、effects、determinism、evaluation、cache、retry、cancellation、timeout 和 InstructionSpec。
- effect 节点声明 attributed capability requirement，并用 RequirementBinding 把 config slot 绑定到 target/credential；config 不能自授 authority。
- 可路由失败同时声明 ErrorSpec 和 error output。预期业务结果用正常 exec output，不滥用 failure。
- StatusEvent 是不可连线观察事实，不拿来控制分支。

## 创作

- title/description/category/tag/icon 和端口文案只放 Authoring，不改变 machine semantics。
- 复杂输入优先由类型的 Editor Adapter 或 schema projection决定；前端不得根据节点名猜控件、端口类型或 capability。
- 新类型/节点必须通过 Catalog seal/open、type × capability closure、Projection/Compiler parity 和对应用户旅程。
