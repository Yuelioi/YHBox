# Compile editable sources into content-addressed programs

Yotta 将 `WorkflowSource 3.1`、`CompileResult` 和 `ProgramSnapshot 3.1` 作为三种不同事实：所有调用方把 raw Source 交给唯一 Compiler，只有成功编译的不可变 Program 才能进入执行队列。我们选择小型 `CompileDraft` interface，把 phase pipeline 和未来的内容寻址仓库隐藏在实现内；拒绝继续让 runtime 按 workflow/container ID 读取、Normalize 或临时编译，因为那会使排队后的编辑改变实际运行内容。

Source、Catalog 与 Program 分别用版本化 domain separation 的 SHA-256 标识，JSON 在哈希前按 [RFC 8785 JCS](https://www.rfc-editor.org/rfc/rfc8785.html) canonicalize；digest 文本只接受 `sha256:<lowercase hex>`。Program artifact 包含 source/catalog hash、compiler build、implementation set、node locks 和按 graph/node/requirement attribution 的 sealed Capability Plan，只有 Compiler 能 seal；持久化和 runtime strict-open 时必须提供可信 Catalog 与 expected compiler digest，并重验 canonical bytes、结构、binding、capability 与 hash。Hash 证明内容完整性，不代表签名、来源认证或权限授权；运行授权来自另行绑定该 Program/plan/Run 的 Run Grant。
