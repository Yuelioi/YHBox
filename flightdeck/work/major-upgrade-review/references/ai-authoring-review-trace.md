# AI authoring change review 与 redacted trace

Status: completed (c71cc19f)

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

无。Agent runtime 已由 d22b5bd5 完成；eval upgrade gate 已由 cfa12703 完成。

## Verification

c71cc19f 建立 internal/aiauthoring bounded native tool loop、Application opaque PreparedPatch/CommitPreparedPatch boundary 与 AIService proposal lifecycle。所有 authoring tools 都是 pure proposal；accept 时验证 base revision/hash 并只提交已审查的 exact candidate。

AuthoringReview 记录 normalized changes、compiler diagnostics、capability/credential/target delta、risks、usage 与 redacted trace；敏感 input 只记录 trust class/digest/size。eval candidate 已加入 Authoring PromptManifest/ToolSet，旧 evidence 自动 stale。

task check 全绿：global coverage 65.8%，internal/ai 74.1%，internal/aiauthoring 62.5%，frontend 28 files / 106 tests，Wails 14 services / 95 methods / 109 models。真实 Windows Wails/WebView smoke 验证 AI review panel 可达且不遮挡 canvas。

## Out of scope

- 绕过 typed patch 的文件/JSON 写入。
- 自动批准危险 capability。
- 长期聊天 Session 与云端 trace 上传。

## Result

Completed in c71cc19f。AI authoring 未经 accept 不产生 durable mutation；权限扩大需要显式用户确认；reject 与 revision conflict 均 fail closed，可从 durable source 重新提案。
