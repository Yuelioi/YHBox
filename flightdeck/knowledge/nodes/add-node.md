---
kind: checklist
summary: "新增 3.1 节点的全链路：类型/能力 → Node Contract → implementation lock/adapter → Projection → Compiler/admission → 用户旅程。"
activation: action
read_when: "新增或修改一个 3.1 node kind、端口、配置、错误、状态、类型、capability 或 runtime adapter 前"
recheck_when: "Node Contract、internal/nodes 组装、noderuntime installation、Projection、frontend catalog 或验证入口变化后"
---
# 新增 3.1 节点 checklist

## 1. 先证明类型与能力闭包

- 复用精确 TypeRef；新增类型时同时定义 schema、representation/carrier、authoring 和 type × capability coverage。
- effect 节点先定义 capability/operation/target kind/scope/risk/consent；不要先写 runtime 再补权限。
- 确认新能力能从同一 installation manifest 投影到 Host Profile；禁止在 appbootstrap 增加第二套手工 capability switch。

## 2. 在 `internal/nodes` 定义并 seal contract

- 稳定 `nodeTypeId` URI，不带产品版本；独立 SemVer `version`。
- config 使用严格 JSON Schema 2020-12，unknown field fail closed；secret 只用 credential slot。
- data/exec/error port 分频道声明，ID 稳定且使用 lowercase/kebab 词汇；纯数据节点没有伪 exec pin。
- 填完整 ExecutionSpec、InstructionSpec、ErrorSpec、StatusEvent、StateAccess、RequirementBinding 和 ABI。
- presentation 只放 title/description/category/tag/icon/editor adapter，不把展示字段混进语义。

## 3. 绑定实现

- 用 `BuiltinDefinition` 把 exact Contract 绑定 implementation entrypoint/version/conformance。
- 在 `internal/noderuntime` 安装匹配 implementation lock 的 adapter；不能按 node title/kind 猜实现。
- adapter 输入/输出必须按 pinned type reopen/reseal；effect 在返回前恰好记录一次真实 AdapterAction。
- 可恢复失败返回 contract-declared `NodeFailure{Code, Output}`；配置、契约和内部错误不得伪装成业务失败分支。

## 4. 创作面

- `internal/nodeauthoring` 从 contract 自动投影端口、类型、控件、target/credential 和 availability。
- zh/en i18n 补节点、端口、配置和错误文案；特殊值编辑使用具名 Editor Adapter，不在 Vue 按 node ID 写大 switch。
- 节点目录、搜索、拖线候选和 Inspector 必须消费同一 Projection。新增资源/State/Target 引用时使用可搜索 picker，不使用全量下拉。

## 5. 验证

- contract seal/open、semantic digest 和 implementation lock。
- type × capability closure、Projection/Compiler parity 和 config validator。
- adapter conformance：真实依赖边界，不只测 stub 调用。
- effect 节点至少一条从 Source → compile → admit → adapter → journal 的集成。
- 属于录制、资产、Windows/ADB 的节点，还要进入对应 Stage 黄金用户旅程；真机 smoke 未完成不能把能力标 completed。

开发中只跑继续工作需要的定向测试；多个相邻节点/adapter 完成后在 Stage 末统一 `task check`、build 和真实宿主验收。
