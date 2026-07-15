---
title: 原型验证 Schema 驱动的节点编辑体验
label: wayfinder:prototype
parent: ../map.md
status: closed
assignee:
blocked_by:
  - define-data-types-and-value-envelope.md
  - define-node-contract-metaschema.md
---

# 原型验证 Schema 驱动的节点编辑体验

## Question

Authoring Projection 应提供怎样的小 interface，才能由 Node Contract 与 Data Type schema 统一生成画布端口、参数控件、类型/约束提示、默认值解释、错误、capability/platform badge、示例和帮助，并让区域选择、代码编辑、AE/UE 对象选择等 Editor Adapter 不再拥有节点语义？

## Resolution

采用 `yotta.node-authoring-projection` 3.1 canonical artifact：生成器必须绑定一个精确 Catalog hash，并完整、逐一匹配 Catalog 中的 Data Type、Capability Definition 与 Node Contract。投影是可丢弃 cache；strict open 必须从可信输入重新生成并要求 canonical bytes 完全一致，不能接受前端或插件修改过的投影。

投影向编辑器提供以下封闭事实：

- data port 的 binding、carrier、完整 TypeExpression、引用类型、representation 与 lifecycle；exec/error/status 仅按 contract 的显式 signal 输出。
- config field 的 control、required、`hasDefault`、examples、deprecated/readOnly/writeOnly、结构与约束。JSON Schema default 只显示提示，绝不自动写入 Source。
- execution class、availability、Editor Adapter、稳定 errors，以及 capability 的 operation、scope、target/credential slot、target kind、risk 与 consent。
- 标量 schema 投影为通用控件；复杂 object/list 显式使用 JSON，或进入 contract 指定的 Yotta 内置 Editor Adapter。Adapter 不拥有端口、类型、权限或执行语义，第三方包也不能注入前端 JavaScript。

生产 Inspector 直接读取生成的 TypeScript projection，显示可见 label/caption/constraints，并用稳定 ID 与 `aria-describedby` 关联控件；类型生命周期、runtime carrier、target 与 capability 风险以紧凑状态和明文细节展示。Markdown 文档消费同一 projection，因此 Concat 只有两个 data input 和一个 data output，不会再出现虚构 `out`。

研究依据与 UI 映射见 [Node Authoring Projection 3.1 research](../../../research/authoring-projection-3.1.md)。
