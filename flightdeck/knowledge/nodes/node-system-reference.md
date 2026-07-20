# 3.1 节点系统速查

## 源码

| 目标 | 权威位置 |
| --- | --- |
| 类型与关系 | `internal/datatype` |
| capability | `internal/capability` |
| Node Contract | `internal/nodecontract` |
| 内建定义/组装 | `internal/nodes` |
| machine Catalog | `internal/nodecatalog` |
| Authoring Projection | `internal/nodeauthoring` |
| runtime adapters | `internal/noderuntime` |
| Source schema/authoring | `internal/workflow/schema`, `internal/workflow/authoring` |
| Compiler/Program/scheduler/debug | `internal/workflow/compiler` |
| Run facts | `internal/run` |

## 当前目录与生成物

- `task nodes`：由当前 sealed builtins/Projection 生成 Markdown 文档。
- `task nodes:catalog`：生成 machine Catalog。
- `task nodes:authoring`：生成 Authoring Projection。
- `task contracts:check`：拒绝 Workflow/Node/Data/Projection 等 tracked contract 漂移。

不要手抄节点总数、端口全集、Wails RPC 数量或类型颜色表；它们是易变观测值，不是契约。需要当前值时运行生成入口或查询 Application 持有的同一 Projection。

## 数据与信号

- data edge 按精确 TypeExpression 编译；pull 节点按需求值，push 节点的输出只在所属 activation 中消费。
- exec/error edge 是有序 signal route；error 不是普通 exec 别名。
- status event 只进 Run journal/debug UI，不是画布 handle。
- BlobRef 是 durable 值；Stream/ResourceRef 只能在 Run lease 内流动并要求显式 ResourceLeaseBinding。
- State 是 Workflow 声明的 typed Run slot；StateRead/Write 的实例类型由所选 slot 专化。

## 旧知识判别

凡提到 `internal/node`、`nodepkg.Spec`、Runnable/RegionRunner/Evaluator、ContainerFlowNode、`config.capture`、GetVar/SetVar、goja 节点函数或旧 `task nodes:pins` 的资料，描述的是 3.0 legacy 栈，不适用于现行实现。
