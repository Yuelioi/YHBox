---
kind: note
summary: "3.1 节点由版本化 Node Contract、Catalog implementation lock、Authoring Projection 和唯一 Program runtime 组成；旧 Spec/Registry/Container dispatch 已删除。"
activation: action
read_when: "第一次进入节点系统、设计节点/类型/能力、修改 Catalog/Compiler/adapter，或需要判断旧节点知识是否仍适用时"
recheck_when: "Node Contract、Catalog、Authoring Projection、Compiler instruction、noderuntime installation 或 Node Package ABI 改动后"
---
# 3.1 节点系统架构

现行链路只有一条：

```text
Data Type + Capability Definition
        ↓
versioned Node Contract → sealed Catalog + implementation lock
        ↓                              ↓
Authoring Projection          Workflow Source → Compiler → Program
                                                ↓
                                     admitted installed adapter
```

旧 `internal/node`、`nodepkg.Spec`、包级 `init/Register`、Runnable/RegionRunner/Evaluator、Container dispatch、`config.capture`、旧 Expr/Var/Script binding 和第二 debugger runtime 均已删除，不得作为新增节点范式。

## 模块所有权

- `internal/datatype`：TypeRef、schema、representation、carrier、traits 和显式类型关系。
- `internal/capability`：capability definition、requirement、scope、target/credential slot 和 sealed plan。
- `internal/nodecontract`：端口、execution、instruction、errors/status、state access、config validator、ABI 和 NodeRef。
- `internal/nodes`：显式组装内建 Data Type、Capability、Node Contract、implementation lock 和 conformance metadata。
- `internal/nodecatalog`：精确 NodeRef ↔ implementation binding 的不可变 snapshot。
- `internal/nodeauthoring`：从 sealed Catalog 派生类型、控件、端口、availability 和 target binding；前端不解释 raw contract。
- `internal/noderuntime`：按 implementation entrypoint 安装 adapter，并在调用时只取得 typed inputs、trigger、窄 capability session、action recorder 和 state binding。
- `internal/workflow/compiler`：唯一编译、scheduler、region instruction、error route、state 和 debug 路径。

## 执行分类

Node Contract 显式声明 execution class、pull/push evaluation、determinism、cache、retry、cancellation、timeout 和 effects。普通节点使用 `invoke`；Run root、counted loop、for-each、retry 等由 host instruction lower。纯数据没有 exec port；status event 不可连线；error 与 exec 是独立 channel。

Adapter 不拥有调度。它必须匹配 Catalog implementation lock，返回已声明类型的 outputs 和显式 signal selection；effect 还要记录真实 AdapterAction。没有旧 kind dispatch fallback。

## 版本与身份

`nodeTypeId` 是不含产品发布号的稳定 URI；节点版本是独立 SemVer，语义摘要与 type ID/version 共同构成精确 NodeRef。`3.1` 只表示 artifact format generation，不进入 package、目录或类型名。

## 正确性边界

- Compiler 是图类型、端口、state、instruction 和 capability plan 的最终权威。
- Projection 与 Compiler 来自同一 sealed Catalog；前端不得维护第二套 assignability/capability 表。
- effect authority 只来自 Admission sealed Run Grant；node config、prompt 和 adapter 返回值不能扩大它。
- Source-native GraphCall、debug、schedule 和 MCP authoring 都进入同一 Compiler/Application seam。
