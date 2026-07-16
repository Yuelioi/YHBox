# AI prompt/tool provenance 与 trusted instruction boundary

Status: completed (b674664c)

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

无。ai-native-design-disposition 已由 b25a0c6c 完成。

## Verification

- PromptManifest、ToolSet、StructuredOutput 已使用独立 versioned hash domain，canonical seal/open 严格拒绝 unknown field、非 canonical bytes、digest mismatch 与越界预算。
- GenerateRequest 携带 RenderedPrompt；OpenAI `instructions` / Anthropic `system` 只从 strict-opened manifest 读取，workflow prompt 始终进入 typed untrusted block。
- AI Generate/Extract Node Contract、authoring projection、frontend i18n 与 runtime 已删除任意 `instructions` config；legacy override 编译为 `INVALID_CONFIG`。
- 内置 prompt digest 进入 implementation lock；AdapterAction 只记录 prompt/schema/toolset digest、provider、requested/resolved model、request/response identity 与 usage，不记录原始 prompt、schema、trusted instructions 或 secret。
- 回归覆盖 prompt injection 分类、manifest/toolset strict reopen、unknown field、digest mismatch、字节/数量预算、redaction、实现锁以及 OpenAI/Anthropic wire mapping。
- 2026-07-17 两次仓库根 `task check` 全绿；最终一次 global coverage 65.4%，frontend 27 files / 103 tests，production build 与契约检查通过。

## Out of scope

- Agent tool loop、tool execution。
- eval corpus 与 upgrade gate。
- AI authoring UI 或 change review。
- generic Chat、prompt JSON fallback 与 provider wire 统一。

## Result

b674664c 完成 trusted prompt/tool provenance：运行时不再接受 workflow 任意高权限指令，provider wire、implementation lock 与脱敏 lineage 都绑定 exact artifacts。下一步由 ai-agent-budget-runtime 消费 ToolSet seam。
