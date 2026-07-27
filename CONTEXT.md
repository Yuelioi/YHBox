# Yotta automation

Yotta 是本地优先的可视化自动化工作台。Workflow 是用户首先理解和管理的对象，其他概念只在创作、
连接、触发或查看结果时出现。

## Product language

**Workflow**:
用户可以创建、编辑、导入、导出和运行的自动化。
_Avoid_: Release、Installation、已安装工作流、不可变工作流

**Target**:
Workflow 实际操作的当前设备对象或服务，例如应用、窗口、浏览器、Android 设备、AI 模型或 HTTP 服务。
_Avoid_: Provider installation、adapter installation

**Schedule**:
让一个或多个 Workflow 按时间、快捷键或启动事件自动运行的可选触发配置。
_Avoid_: Job、计划任务实例

**Resource**:
Workflow 使用的图片、Macro、InputClip 或其他创作素材。Resource 可以跟随一个 Workflow，也可以保存在
本机资源库供多个 Workflow 使用。
_Avoid_: Artifact、Blob

**Run**:
Workflow 的一次执行及其状态、时间线和结果。
_Avoid_: Execution installation、运行实例对象

## Authoring language

**Graph**:
Workflow 中由节点和连线组成的一张逻辑图。
_Avoid_: Canvas document、flow definition

**Main Graph**:
Workflow 开始运行时进入的 Graph。
_Avoid_: Entry workflow

**Subgraph**:
Workflow 内可被其他 Graph 调用的可复用 Graph。
_Avoid_: Nested workflow、child workflow

**Node Type**:
一种节点的定义，描述它接受什么、产生什么以及可配置什么。
_Avoid_: Node package instance

**Node**:
Graph 中对一种 Node Type 的具体使用。
_Avoid_: Node instance object

**Snippet**:
用户保存并可再次插入的一组节点、连线和相关创作信息。
_Avoid_: Mini workflow

## Local configuration

**Target Slot**:
Workflow 用来表达目标用途的逻辑名称，不包含某个用户的应用路径、窗口身份或设备地址。
_Avoid_: Installation ID

**Target Binding**:
当前设备把 Target Slot 连接到一个具体 Target 的设置。
_Avoid_: Workflow-local path、release configuration

**Credential Binding**:
当前设备把 Workflow 的逻辑凭据需求连接到本机凭据的设置。Credential Binding 不包含在 Workflow 或
导出文件中。
_Avoid_: Embedded secret、workflow credential

## Durable language

**Workflow Source**:
一个 Workflow 的可编辑、可持久化表示，拥有稳定 Workflow ID 和递增 revision。
_Avoid_: Release payload、installation source

**Diagnostic**:
创作或运行前发现的问题，包含问题位置和用户可采取的修复动作。
_Avoid_: Admission verdict、policy result
