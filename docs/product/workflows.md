# Workflows and authoring

Workflow 是 Yotta 的主要创作与运行对象。当前 GUI 的主入口是 Workflow 列表和编辑器；后端的持久事实是
`internal/workflow/schema.WorkflowSource`，而不是 Vue Flow 状态、已编译 Program 或某次 Run。

## Workflow Source 保存什么

一份 Source 包含以下可移植内容：

| 内容 | 代码中的含义 |
| --- | --- |
| Workflow metadata | 稳定 ID、名称、描述、分类、标签与时间戳 |
| revision | 每次成功提交后递增，用于 compare-and-swap 冲突保护 |
| derivedFrom | 把 immutable Release 复制成可编辑 Source 时的精确 provenance；不包含本机 authority |
| main graph / subgraph | 主入口图与可复用子图；graph call 显式连接接口 |
| node / edge | 节点引用、配置、输入绑定、位置，以及 data/exec/error 三种连线 |
| state variable | Workflow 级、精确类型的状态声明和默认值；每个 Run 独立实例化 |
| resource | 随 Workflow 携带的 image、macro 或 input-clip |
| target profile/default | 目标需求的可移植 schema、发现提示和逻辑 slot 默认值 |
| credential requirement | 只描述凭据用途和逻辑 slot，不保存本机 secret |
| package dependency | 精确 package、manifest digest 与 NodeRef 依赖 |

Source 不保存本机可执行文件路径、临时窗口句柄、当前设备连接、浏览器 session、API key、运行期 Stream/
HeldInput handle 或 Program cache。这些对象分别属于本机配置、secure store 或单次 Run。

机器可读的当前 Source schema 位于 `contracts/workflow/current/`，由 Go 类型生成；不要根据本页反向实现 parser。

## 图与节点

每个 Workflow 恰有一个 entry graph。Graph 可以是 `main` 或 `subgraph`，并包含 node、graph call、edge、接口
port、entry/exit 和 annotation。三种 edge channel 的语义不同：

- `data` 传递精确 Data Type 的值。
- `exec` 选择正常控制流。
- `error` 路由节点声明的可处理失败；未路由失败会终止 Run。

Node Type 由版本化 Node Contract 定义端口、配置 schema、执行语义、错误、状态、Target/capability 需求和
implementation lock。编辑器消费同一 sealed Catalog 派生的 Authoring Projection；Compiler 才是连接、类型、
控制流和依赖是否有效的最终权威。

当前节点清单从实现生成：

```powershell
task nodes
task nodes:catalog
task nodes:authoring
```

`Snippet` 不是一张小 Workflow，也不是节点组。当前 `internal/services/snippet.NodeTemplate` 只保存一个可再次
插入的 Node 模板及其 config、bindings、disabled 状态、metadata 和快捷键。多节点逻辑应使用 Subgraph。

## 修改与冲突

GUI、MCP 和 AI authoring 不直接覆盖整份 Source。正式修改路径是：

```text
base revision + typed commands
          │
          ▼
authoring reducer ──> strict Source validation ──> compile/check
          │
          ▼
revision CAS publish
```

Typed command 覆盖 metadata、Target default、state、node/config/binding、resource、connection、graph/interface、
call、annotation、reroute 和 collapse 等编辑。提交时 base revision 已过期会返回冲突，调用方必须重新读取当前
Source；不能用 last-write-wins 掩盖并发修改。

AI authoring 先生成可审阅 proposal，用户接受后仍走同一 patch/commit 边界。前端 undo/selection/camera 是
编辑会话状态，不会自动成为 Source 事实。

## 检查、运行与导入导出

- `CheckDraft` 检查尚未持久化的 Source JSON；`CompileSource` 编译 Catalog 中的当前 revision。Source、NodeRef、
  package、graph、port、type 或 config 问题在这里成为 compiler diagnostic。
- `PreviewRun` 复用 stored-source compile 并返回 Program hash 与冻结的 capability plan；它不读取当前 Target/
  credential/provider，不做 admission，也不创建 Run。
- `StartRun` 才取得当前 Configured Target generation 并完成 capability admission。调用方用
  `ClassifyRunStart` 区分 workflow invalid、Target/credential required 与 persistence/system failure。
- 普通 Run 与 Debug Run 使用同一个 immutable Program 和 executor；Debug 只增加 breakpoint/pause/step/
  continue 控制。
- Workflow bundle 是 Source 与其可移植 Blob 的交换边界。导入先 inspect，再创建或以 expected revision/hash
  替换；导出不会夹带本机 settings 或 secret。
- 无法作为当前 Source 打开的持久记录进入 recovery/quarantine 面，由用户修复或删除，不会静默退回旧 parser。

运行链的实现边界见[Workflow runtime](../architecture/runtime.md)，本机 Target 与 Resource 的区别见
[Targets, capabilities and resources](targets-and-resources.md)。
