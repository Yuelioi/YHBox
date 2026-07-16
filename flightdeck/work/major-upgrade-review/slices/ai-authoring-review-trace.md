# AI authoring change review 与 redacted trace

Status: blocked

## Outcome

AI authoring 作为 typed Application client 在有限预算内执行 search/inspect/patch/compile/preview 闭环；用户在应用 patch 或运行前可检查 graph diff、diagnostics、permission delta 与完整脱敏 provenance。

## Completion criterion

- AI authoring loop 只调用 catalog search/describe、workflow inspect/apply-patch/compile/explain/preview，不读取或替换整图 JSON。
- 每轮受 iteration/tool/token/time budget；revision conflict、diagnostic stall 与 permission expansion 有明确终态。
- review artifact 对比 base/new revision，列出 normalized domain changes、compiler diagnostics、capability/credential/target delta 和 unresolved risk。
- trace 关联 model profile、PromptManifest、ToolSet、schema、provider request、approval、patch、compiler 与 Run outcome；敏感 input 只保留 trust class/digest/size。
- UI 提供 accept/reject/retry 与审计查看；未经接受不得发布 mutation 或 admission authority。
- 端到端 tests 覆盖无关节点保护、冲突、injection、权限扩大、redaction 与恢复；task check 和 WebView smoke 全绿。

## Blocked by

ai-agent-budget-runtime、ai-eval-upgrade-gate。

## Verification

MCP/Application typed authoring substrate、compiler diagnostics、capability preview 与 Run AdapterAction 已存在；没有 AI loop、review artifact/UI 或贯穿 prompt/tool/patch/run 的 redacted trace。

## Out of scope

- 绕过 typed patch 的文件/JSON 写入。
- 自动批准危险 capability。
- 长期聊天 Session 与云端 trace 上传。

## Result

Blocked。
