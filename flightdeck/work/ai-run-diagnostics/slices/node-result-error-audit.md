# 全节点结果与错误契约审计

## Outcome

为所有内置 Node Type 建立可机械检查的结果/timeout/failure 契约清单，定义面向用户与 AI 的稳定错误 ID、typed
参数、内部 cause 和本地化规则，并按节点族给出迁移顺序。

## Scope

- Node Contract 的 exec output、error output、status event 与数据输出。
- Node runtime、configured target adapter、provider、Application/Service 到前端 transport 的错误映射。
- 编译期 Diagnostic、Run terminal failure、节点 error route、正常 timeout/exhausted 结果的区别。
- 前端 i18n、错误展示、时间线导出、MCP/AI 读取和旧版 EXE/契约不兼容场景。

## Questions

1. 哪些节点把 timeout/not-found/exhausted 表达为正常业务结果，分别应返回哪些结构化证据？
2. 哪些节点可能把包名、adapter 名、Go cause 或英文实现消息泄漏给用户？
3. 稳定错误 ID 的命名空间、参数类型、敏感字段和 fallback 文案如何定义？
4. Error Envelope 应位于哪个深 module 的 interface，如何同时供 Timeline、前端、MCP 和 AI 使用？
5. 旧 Program、旧 EXE、Node Contract mismatch 与 package incompatibility 如何映射为可操作且可翻译的错误？
6. 如何通过 schema/test/lint 保证新增错误必须注册 ID、参数和翻译，而不是依赖人工 review？

## Current

源码审计确认 `NodeFailure` 只有 code/output/cause，缺少 typed params；runtime 与 adapter 大量使用硬编码英文 cause；
前端存在按英文错误子串分类；不同节点的 timeout/exhausted 语义不一致且没有统一证据模型。完整证据与行号见
[错误契约源码审计](../references/error-contract-audit.md)。

已有代码也提供可复用基础：前端 transport 能归一化 `{code, params}`，Compiler Diagnostic 已采用稳定 code/params，
MCP typed patch 与 catalog contract 可承载机器可读修复。缺口集中在 runtime/adapter/service seam 没有统一 Problem。

## Candidate interfaces

面向用户、Timeline、MCP 和 AI 的失败投影统一为：

```text
Problem {
  id              // 稳定、分层命名，例如 runtime.package.incompatible
  params          // 由 id 注册的 typed schema 约束
  category        // validation | target | adapter | node | infrastructure
  retryable       // 机器可用的重试提示
  location        // 可选 workflow/graph/node/attempt/field/port
  supportId       // 可选，与内部日志 cause 关联，不暴露 cause 文本
}
```

内部错误保留 `cause` 用于 wrapping、`errors.Is/As` 和日志，但 service/transport 只能返回 Problem。前端不得通过
substring、正则或包/adapter 名推断错误类型；翻译键由 `problem.<id>` 派生，缺翻译时显示稳定 ID 与 support ID。

正常的条件未满足统一为 Node Outcome Evidence，而不是 Problem：

```text
NodeOutcomeEvidence {
  outcomeId       // found | timeout | gone | exhausted | unchanged ...
  metrics         // 节点契约注册的有界 typed metrics
  observations    // 可选的有界候选/状态事实
}
```

每个 Node Type 必须声明：成功 outputs、非错误 outcomes、error route problems、status evidence 和敏感参数策略。

## Mechanical gates

- Error ID 注册表生成 Go/TypeScript schema 与 i18n key 清单；重复、未注册或缺翻译直接失败。
- 禁止 service transport 直接返回未分类 `err.Error()`；未知 cause 归一为 `system.unexpected` + support ID。
- 禁止前端用 message substring/regex 分类；只允许检查稳定 id/category。
- Node catalog conformance test 遍历所有内置 Node Type，验证 timeout/exhausted outcome 与 error output 的声明。
- Adapter failure mapping table 必须覆盖每个 adapter code 到公开 Problem；原始 command/path/device message 按参数策略脱敏。

## Next

None. Problem interface 已贯通 Wails、async result、Automation health、MCP、NodeFailure、Run 与前端渲染。
