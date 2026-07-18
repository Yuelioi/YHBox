---
kind: checklist
summary: "3.1 数据流由 typed data edge、独立 exec/error signal route、activation-scoped push output、pull evaluation 和显式 typed State 组成。"
activation: action
read_when: "设计节点输入输出、连接 data/exec/error edge、跨图传值、State 复用或排查数据连不上/值过期时"
recheck_when: "Compiler data plan、scheduler activation、GraphCall interface、StateAccess 或 ResourceLeaseBinding 改变后"
---
# 3.1 节点数据流

## 五种不同语义

1. **Data edge**：精确 TypeExpression 的值依赖；Compiler 验证类型和 data DAG。
2. **Exec route**：有序控制信号。
3. **Error route**：携带结构化 routed failure 的独立频道，不是普通 exec 别名。
4. **Run State**：Workflow 声明的 typed slot，用于显式命名复用；每个 Run 独立。
5. **GraphCall interface**：跨图 typed inputs/outputs 与 exec/error exits；不能靠全局变量穿透边界。

纯数据/pull 节点在消费者需要时求值；effect/control 等 push instruction 只执行一次，其 data output 在所属 activation 中可供已编译的下游读取。旧 Container 的全局 held-output map、同名 exec-data 注入和 `config.capture` 已删除，不能作为当前连线模型。

数据输入的来源在 Source 中必须显式：edge、literal/default、GraphCall binding 或 StateRead。Runtime 不根据端口名、颜色、节点标题或前一次运行猜值。输出必须按 Program pinned TypeRef reopen/reseal；runtime-only carrier 只能通过 ResourceLeaseBinding 借用。

需要跨多个位置复用一个 durable 值时，使用 Promote to State 或显式 StateWrite/StateRead。不要为了省一个节点加入 ambient variable、前端缓存或隐藏 conversion。类型变化必须预览全部引用并重新编译。

GraphCall lowering 会为每个调用点分配独立 runtime identity并保留 Source provenance；跨图数据必须经过声明的 callable interface。comment/reroute 只属于 authoring metadata，不参与数据流。

排查顺序：Source edge/channel → exact port/type → Compiler diagnostic → activation/provenance → adapter output reseal → State/lease lifecycle。不要从前端颜色或静态标题反推类型。
