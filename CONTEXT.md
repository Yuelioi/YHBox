# Yotta V4 domain

Yotta 是本地优先的工作流工具。用户应当只需要理解五个一级概念：Workflow、Target、Schedule、
Resource 和 Run。编译、摘要、provider 与存储代际是实现细节，不形成第二套产品流程。

## Product language

**Workflow**

用户可以创建、编辑、导入、导出和运行的自动化。文件导入后就是普通本地 Workflow，不产生只读类型、
安装实例或更新状态机。

**Target**

工作流实际操作的本机对象或服务，例如 AI 模型、HTTP 服务、Windows 应用、窗口、浏览器或 Android
设备。Target 配置属于当前设备，保存后立即可用。

应用路径、窗口身份、设备地址、API endpoint 和校准信息都属于 Target，不属于 Workflow 文件。不同用户
可以用同一个 Workflow 绑定不同 Target。

**Schedule**

对一个或多个 Workflow 的本地触发配置。Schedule 持久引用 Workflow ID，可使用手动、定时、一次性或
热键触发。

**Resource**

工作流使用的图片、Macro、InputClip 或其他素材。Resource 可以归属于一个 Workflow，也可以保存在本机
资源库供多个 Workflow 复用。

**Run**

Workflow 的一次执行及其结果。Run 记录排队、执行、成功、失败和取消状态，并保留必要的节点时间线与
输出引用。

## Authoring and execution

**Workflow Source**

Workflow 的可编辑 JSON 表示。它拥有稳定 Workflow ID 与递增 revision；格式版本独立于 Yotta 产品版本。

**Graph**

Workflow 内的一张节点图。主图是运行入口；子图用于复用局部逻辑。

**Node Type**

一种可用节点的定义，描述输入、输出、配置和执行行为。

**Node Instance**

Graph 中对 Node Type 的一次具体使用。

**Data Type**

节点端口交换值的语义类型。显式转换必须在 Graph 中可见。

**Program Snapshot**

Workflow Source 编译后的内部执行快照。它按内容寻址，只用于保证一次 Run 执行确定的编译结果，不是
用户要管理的对象。

**Diagnostic**

编译或运行前发现的问题，包含稳定 code、位置和可执行修复提示。缺少 Target、Credential 或 Node 时直接
报告缺项，不引入授权流程。

## Local configuration

**Target Slot**

Workflow 节点声明的逻辑目标名，例如 `desktop`、`browser` 或 `model`。Target Slot 不包含某个用户的
应用路径或设备身份。

**Target Binding**

当前设备把 Target Slot 绑定到一个具体 Target 的设置。修改绑定不改写 Workflow Source。

**Credential Binding**

当前设备把逻辑凭据槽绑定到安全存储中的凭据。Workflow、导出包和前端响应不包含 secret。

**Global Asset**

独立于单个 Workflow 的本机 Resource。

**Workflow Resource**

跟随 Workflow 导入和导出的 Resource。

## Internal boundaries

**Application**

GUI、CLI、MCP、Schedule 和快捷入口共用的工作流命令入口。所有运行都调用同一条
`StartRun(workflowID)` 路径。

**Compiler**

把 Workflow Source 与当前 Node Catalog 转成 Program Snapshot，并返回完整 Diagnostic。

**Node Runtime**

执行 Program Snapshot 中的节点。Windows、浏览器、Android、AI 与 HTTP 能力通过各自 adapter 接入。

**Blob Store**

保存图片等不可变大对象，并通过摘要引用；物理路径不进入 Workflow 身份。

**Product Version**

Yotta 应用版本，例如 `4.0.0`。它不等于 Workflow 格式、节点协议或存储布局版本。

## V4 exclusions

V4 核心不包含 Workflow Release、Workflow Installation、Publisher、来源信任、安装计划、离线依赖包、
执行 consent 或远程禁用模型。未来只有在真实用户场景出现并能证明价值时，才可以作为独立可选能力重新
设计，不能污染本地 Workflow 的创建、导入、编辑、运行和计划路径。
