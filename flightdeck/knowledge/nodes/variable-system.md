---
kind: note
summary: "3.1 变量是 Workflow 声明的 typed Run State；StateRead/Write/Metadata 由 slot 专化，每个 Run 隔离，不再有 Container local/global/auto VarStore。"
activation: action
read_when: "修改 Run 状态、StateRead/Write、Blackboard、状态拖拽、状态改型、跨图引用或需要命名复用值时"
recheck_when: "Workflow state schema、StateAccessSpec、Compiler runState、Blackboard UX 或 durable state policy 改变后"
---
# Typed Run State

Workflow Source 声明状态 slot 的稳定名称、精确 TypeRef 和 initial value。Compiler 把声明冻结进 Program；Executor 为每个 Run 创建独立 state，两个 Run 不共享值。3.0 的 Container local/global/auto scope、GetVar/SetVar/IncVar、`$name` 和 ambient VarStore 已删除。

- `StateRead<T>`：pull-data，输出 `result:T`。
- `StateWrite<T>`：push effect，输入/输出 `T`，显式改变当前 Run state。
- `StateMetadata<T>`：读取 revision 和 changed-at，用于变化判断而不泄漏动态类型。

节点 instance 通过 config 中的 `variable` slot 绑定 `StateAccessSpec`。选择 slot 后泛型 `T` 由声明专化；连线不能反向改变状态类型。Compiler 拒绝缺失 slot、类型不满足、runtime-only carrier 和非法初值。

创作体验应提供搜索、类型/作用域显示、从 Blackboard 拖出 Read/Write、从输出 Promote to State，以及改型前的全引用影响预览。常用原子操作（例如 increment）可以作为 typed convenience node/authoring command，但不能恢复动态 VarStore。

State 适合单 Run 内命名复用和显式跨图 interface binding；跨 Run 持久状态需要独立、明确的 durable store/capability 设计，不能偷用当前 Run State。
