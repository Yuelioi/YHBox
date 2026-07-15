---
title: 定义 Node Contract 3.1 元模式
label: wayfinder:grilling
parent: ../map.md
status: closed
assignee:
blocked_by:
  - define-data-types-and-value-envelope.md
---

# 定义 Node Contract 3.1 元模式

## Question

Node Contract 应如何完整声明稳定身份、配置 schema、静态与实例化端口、Execution Class、determinism、capability、errors、retry、presentation annotations、文档、示例、实现锁与 contract hash，使 executable、authoring 和 docs projection 可从同一事实生成且不会互相污染身份？

## Resolution

已接受。Node Contract 3.1 使用一个 strict、canonical、可内容寻址的文档作为 Node Type 的唯一机器事实；Catalog Binding、Effective Node Contract 和 Authoring Projection 是它的派生物，不得反向补语义。

### 1. 身份与版本

- 文档固定 `format: "yotta.node-contract"`、`version: "3.1"`；解析器不猜版本，不接受 unknown fields、非 canonical JSON 或旧格式。
- `NodeRef` 同时包含绝对且以独立 `/vN` 结尾的 `nodeTypeId` 与 `semanticDigest`。产品 3.1 不进入节点身份；首版摘要域固定为 `yotta/node-contract-semantic/v1`。
- `semanticDigest` 对 canonical semantic document 计算，排除摘要字段自身、全部 authoring/docs annotations 和安装时实现制品锁。语义改变必须发布新 `nodeTypeId` 版本；只改标题、翻译、icon、分类或帮助文案可保留 NodeRef。

### 2. semantic document

semantic 至少包含且只包含：

1. `configSchema`：Draft 2020-12 离线 bundle 与唯一 root `$id`；禁止联网 `$ref`，执行 byte/depth/node/reference budget。validation constraints 与 compiler-applied defaults 属于语义；title/description/examples/widget 等展示信息不得藏进 machine schema。
2. 五个必填端口数组：`dataInputs`、`dataOutputs`、`execInputs`、`execOutputs`、`errorOutputs`。不存在的端口必须编码为 `[]`，不能缺字段、写 `null` 或由前后端补默认 `out`。`statusEvents` 是顶层、不可连线的 Run 观察事实声明，不属于 PortSet。
3. Data port 使用规范化 Type Expression；input 的 `required` 只描述 Binding State，不把类型隐式变 nullable。default 必须是显式 canonical JSON binding，并由 Compiler 的 default phase 应用。
4. control/error port 只描述对应通道，不复用 Data Type 字符串。端口 ID 是稳定 machine identity；label、颜色和排序提示属于 authoring。status event 以稳定 code 与 progress/waiting/connection category 声明，由 journal 观察，不参与分支选择。
5. `execution`：独立声明 Execution Class、effects、determinism、evaluation/cache、retry safety、cancellation 与 timeout contract。pure-data 必须 effects、exec/error 与 status event 声明为空；确定性不能从 pure 推断。
6. `capabilities`：稳定、排序、去重的 capability requirement ID；精确 grant/target planning 由后续 Capability ticket 定义。
7. `errors`：稳定 code、category 与 retry hint 的封闭声明；运行时不得返回 contract 未声明的业务错误 code，host/infrastructure error 使用宿主命名空间。
8. 可选 `instanceResolver`：缺失表示 static contract；存在时必须 pin resolver ID、semantic digest 与最大 effective-port budget。resolver 是 `canonical config + frozen contract/catalog -> Effective Node Contract` 的纯确定函数，不得读取网络、文件、时间、随机数或 mutable registry。
9. `implementationABI`：只声明可接受的 builtin/WIT/process ABI surface。具体 implementation kind、entrypoint、artifact digest 和 package version 不进入 portable Node Contract，而由 Catalog Binding 锁定；Program Snapshot 再复制实际 lock。因此 contract hash 不能冒充 executable implementation identity。

### 3. Authoring 与文档

`authoring` 独立保存 title/description i18n key、category、tags、icon、examples、security/lifecycle help 和内置 `editorAdapter` 引用；它不参与 semantic digest，但进入独立 presentation generation。第三方包不能注入 JavaScript/Vue，adapter 只能引用 Yotta 内置 allowlist。

Catalog generator 必须从同一 sealed contract 产生 machine catalog、TypeScript authoring model、JSON Schema、WIT/Protobuf surface、MCP describe 与 node reference。生成物携带 NodeRef、presentation digest 和 generator version；CI 生成后 diff 必须为空。

### 4. 端口与执行不变量

- `pure-data`：只能有 data ports，evaluation 为 pull；只有 `deterministic + effects=[]` 才能声明可缓存。
- `effect`：至少声明一个 effect 或 capability，默认 retry safety 为 `never`。
- `control`/`region`/`event`：控制语义由 Program lowering 表达，runtime 不按 node kind 猜行为。
- `marker`/`visual`：不得绑定普通可执行 implementation；Compiler 必须 lower 或删除它们。
- 全部端口 ID 在各方向内唯一；静态和 instance-resolved 端口合并后重新执行同一预算、唯一性和 Type Expression 校验。
- Error 不等于 Exec，UI、Compiler、Program 与 runtime 必须保留通道种类；Status 不等于任何连线，它只能成为 Run event。

### 5. 首个 conformance contract

`text.concat/v1` 是首个 golden contract：两个 required string data inputs `a`、`b`，一个 string data output `result`，五个 data/exec/error 端口数组中的控制与错误数组全部为空，`statusEvents=[]`；Execution Class=`pure-data`、effects=`[]`、determinism=`deterministic`、evaluation=`pull`、cache=`per-run`、retry=`never`、capabilities/errors=`[]`。它必须贯通 Catalog 3.1、Source 3.1、Compiler、Program、interpreter、Vue、MCP 与生成文档；任何层出现伪造 `out` 都是 conformance failure。
