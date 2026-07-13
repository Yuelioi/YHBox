# Compile editable sources into content-addressed programs

Yotta 将 `WorkflowSource v3`、`CompileResult` 和 `ProgramSnapshot` 作为三种不同事实：所有调用方把 raw Source 交给唯一 Compiler，只有成功编译的不可变 Program 才能进入执行队列。我们选择小型 `CompileDraft` interface，把 phase pipeline 和未来的内容寻址仓库隐藏在实现内；拒绝继续让 runtime 按 workflow/container ID 读取、Normalize 或临时编译，因为那会使排队后的编辑改变实际运行内容。
