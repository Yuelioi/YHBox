# Yotta Workflow Automation

Yotta 将可编辑的自动化意图编译成不可变程序，再把程序绑定到本机能力执行。可编辑事实、可执行事实和运行事实必须使用不同身份。

## Language

**Workflow Source**:
用户或 AI 可编辑的 v3 自动化文档，带文档身份与 revision，但本身不可执行。
_Avoid_: Container, blueprint, flow

**Compile Result**:
对某一份 Workflow Source 和 Catalog Snapshot 的完整诊断结果；没有 error diagnostics 时才包含 Program Snapshot。
_Avoid_: Validation result, build response

**Program Snapshot**:
Compiler 产生的不可变、内容寻址可执行事实；后续编辑 Workflow Source 不改变它。
_Avoid_: Compiled container, runtime config

**Catalog Snapshot**:
一次编译所绑定的不可变节点目录代际，包含节点 contract 与实现锁。
_Avoid_: Current registry, node list

**Diagnostic**:
Compiler 对 Source 的稳定机器可读判断，以 code、位置、params 和 optional fix 表达；message 只用于展示。
_Avoid_: ValidationError, error message

**Run**:
对一个 Program Snapshot 的一次有身份执行，记录 program hash 与调用绑定。
_Avoid_: Container run, workflow execution
