# AI prompt/tool provenance 与 trusted instruction boundary

Status: blocked

## Outcome

建立可 strict-open 的 content-addressed PromptManifest、ToolSet 与 rendered instruction artifact；只有版本化可信指令能进入 provider system/developer block，workflow、用户、网页、OCR 与 tool result 只能作为 typed untrusted input。

## Completion criterion

- PromptManifest 与 ToolSet 具有 canonical bytes、versioned hash domain、严格 reopen、byte/depth/count budgets 和 unknown-field rejection。
- Prompt render 明确区分 trusted instructions 与 typed user/context/tool-result；运行时 config/data 不能构造或覆盖 trusted block。
- AI Generate/Extract 只引用 manifest/profile/toolset identity，不持久化任意 system/developer 文本。
- provider-native request 与 AdapterAction lineage 记录 model profile、prompt、schema、toolset digest，不记录 secret 或原始敏感内容。
- 旧 `instructions` config 从 Node Contract、projection、frontend 与 runtime 一次性删除，不保留 fallback。
- tests 覆盖 prompt injection、digest mismatch、unknown field、budget、redaction 与 OpenAI/Anthropic wire mapping；task check 全绿。

## Blocked by

ai-native-design-disposition。

## Verification

当前 ModelProfile 与 strict schema 已 content-addressed；`nodes31runtime.aiRequest` 仍把 workflow config instructions 直接升格到 provider 高权限字段，仓库没有 PromptManifest/ToolSet artifact。

## Out of scope

- Agent tool loop、tool execution。
- eval corpus 与 upgrade gate。
- AI authoring UI 或 change review。
- generic Chat、prompt JSON fallback 与 provider wire 统一。

## Result

Blocked，等待 disposition handoff。
