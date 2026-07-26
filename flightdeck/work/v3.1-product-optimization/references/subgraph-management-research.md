# 子图管理调研

## Research Read

研究问题：Yotta 的工作流编辑器怎样让用户可靠地创建、进入、编辑接口、复用、展开和删除子图，同时明确区分“画布上的一次调用”和“可被多处调用的子图定义”。

目标界面是工作流画布、顶部图导航和子图接口面板。用户当前遇到的失败包括：

- 调用节点及内部边界节点的标签与连接点重叠。
- 空子图的入口、出口看得见，却缺少明确的创建和管理动作。
- `Delete` 删除调用节点后，子图定义仍出现在“图列表”，用户会理解为没有真正删除。
- 多个默认名“子图”无法辨认，也看不到定义被哪些调用引用。

约束：

- Yotta 当前 Source 已经采用“定义 + 引用”模型：`WorkflowSource.graphs` 保存 `Graph` 定义，父图的 `Graph.calls` 保存 `GraphCall { id, graphId, ... }` 实例。
- `Graph.inputs / outputs / entries / exits` 是唯一接口事实；内部边界节点和外部调用节点都应只是投影。
- GraphCall 由现有 compiler lower 到同一 Program，不增加第二套节点或 runtime。
- 当前只承诺一个执行入口、多个命名 exec/error 出口；本轮管理设计不暗中扩成多入口。

## Source Matrix

| 来源 | 采用的事实 | 对 Yotta 的启示 |
| --- | --- | --- |
| [Unreal Engine：Collapsing Graphs](https://dev.epicgames.com/documentation/unreal-engine/collapsing-graphs-in-unreal-engine?lang=en-US) | 选区可折叠为命名图；双击调用进入；Input/Output 节点可在 Details 中增添 pin；可 Expand Node 还原。 | 折叠、进入、接口编辑、展开应形成完整闭环，不能只有“折叠”单向操作。 |
| [Unreal Engine：Nodes / Collapsed Graphs](https://dev.epicgames.com/documentation/unreal-engine/nodes-in-unreal-engine?lang=en-US) | 折叠图在独立列表中可打开；父图用 tunnel pins 镜像内部接口；Unreal 明确把仅用于整理且复制时深拷贝的 collapsed graph 与可共享的 macro/function 区分。 | Yotta 的 GraphCall 已是共享引用，UI 必须明确写“调用”和“定义”，不能借用会暗示独占容器的模糊“子图节点”语义。 |
| [Blender 5.0：Node Groups](https://docs.blender.org/manual/fi/5.0/interface/controls/nodes/groups.html) | Make Group 自动生成 Group Input/Output；接口面板支持增删、重命名、排序、复制 socket；连到空心 socket 也可新建接口；Tab 进入、breadcrumb 返回；Ungroup 将内部节点放回父图。 | 接口既要有集中式列表编辑，也可提供画布直连快捷方式；导航要有面包屑；“展开到父图”是重要的可逆操作。 |
| [Blender 5.0：Data-Blocks](https://docs.blender.org/manual/en/5.0/files/data_blocks.html) | 共享定义有用户计数；unlink 一次使用与直接删除 data-block 是不同动作；Make Single User 为一个实例复制独立定义。 | 每个子图定义应显示调用数；删除调用、删除定义、复制定义必须是三个明确动作。 |
| [Node-RED：Subflows](https://nodered.org/docs/user-guide/editor/workspace/subflows) | 支持空建和“选区转子流程”；子流程成为可重复放置的实例；内部工具条直接管理入口/出口；定义可配置名称、说明、外观和实例属性；删除定义会删除全部实例。 | 管理入口应覆盖空建、提取、放置调用和定义属性。Node-RED 的级联删除可以作为显式高级动作，但不适合 Yotta 的默认 Delete。 |
| [Node-RED：Flows](https://nodered.org/docs/user-guide/editor/workspace/flows) | 图可命名、写说明、重排；存在集中列表与快捷切换。 | 图数量增长后需要可搜索列表，而不是只显示同名下拉菜单。 |
| [Unity Shader Graph：Sub Graph](https://docs.unity3d.com/cn/Packages/com.unity.shadergraph%4010.5/manual/Sub-graph.html) | Sub Graph 是被其他图引用的独立定义；Blackboard properties 定义调用节点输入，Output 节点定义输出。 | Graph interface 必须驱动所有 GraphCall 端口；调用实例不能拥有另一份接口状态。 |

## Patterns

### 1. 定义和调用采用两层对象语义

**规则**

- `Graph` 的 UI 名称是“子图定义”。
- `GraphCall` 的 UI 名称是“子图调用”。
- 画布按 `Delete` 只删除选中的调用实例及其连线。
- 只有“子图管理”中的危险操作才能删除定义。

**何时使用**

- 一个行为可能被多处复用、需要统一修复时，多个 GraphCall 指向同一个 Graph。
- 用户只想让某个调用点变成独立变体时，用“复制定义并改用副本”，类似 Blender 的 Make Single User。

**避免**

- 不把“复制调用”实现成深拷贝定义。
- 不让定义工具栏上的垃圾桶和画布 Delete 使用同一文案、同一确认框。

### 2. 接口是可编辑清单，边界节点是同一事实的画布投影

**组件结构**

- 子图右侧面板固定展示：执行入口、数据输入、数据输出、命名出口。
- 每项可重命名；数据端口显示类型；出口显示 exec/error channel。
- 面板提供“添加数据输入 / 添加数据输出 / 添加完成出口 / 添加错误出口”。
- 新接口项先处于“未绑定”状态，再通过拖线或“选择内部端点”绑定。
- 自动推断保留为快捷动作，不能成为创建接口的唯一方式。

**状态**

- `bound`：正常。
- `unbound`：可保存草稿但不可编译，显示就地说明和“选择端点”。
- `in-use`：删除前显示引用调用和连线；后端继续阻止静默破坏。
- `invalid`：类型或 channel 不匹配，就地标红并可定位内部端点。

**实现约束**

- `Graph.inputs / outputs / entries / exits` 始终是 canonical state。
- `WorkflowGraphBoundary` 和 `WorkflowGraphCall` 只消费该接口，不各自保存 pin。
- 当前单执行入口约束应直接体现在 UI：入口是一个固定槽位，不提供“再加入口”按钮。

### 3. 集中式子图管理器替代同名下拉菜单

**最小界面**

- 顶部“图列表”打开可搜索 popover 或侧栏。
- 主图单独置顶；子图定义按名称列出。
- 每行显示：名称、调用数、接口摘要（`1 入 / 2 出口`）、是否存在未绑定或编译错误。
- 行动作：打开、重命名、定位调用、复制定义、删除定义。
- “定位调用”展开调用位置列表：`父图名称 · 调用标签`，点击后跳转并选中。

**命名**

- 新建时必须输入非空显示名；重复名允许但给出提示，并始终保留稳定 graph ID。
- 折叠后直接进入内联重命名。
- 调用默认跟随定义名称；只有用户明确设置实例标签后才不随定义改名。

**导航**

- 双击 GraphCall 进入定义。
- 面包屑不仅显示图名，还保留“经由哪个调用进入”的 UI 上下文，返回时能回到原调用并选中它。
- 从管理器直接打开定义时，面包屑为 `主图 / 子图名称`，不伪造调用路径。

### 4. 删除、展开和复制使用明确的生命周期动作

| 用户动作 | Source 语义 | 默认安全行为 |
| --- | --- | --- |
| 删除调用 | `remove-graph-call` | 删除一个 GraphCall 和关联边；Graph 定义保留。 |
| 删除定义 | `remove-graph` | 仅在引用数为 0 时直接允许；引用数大于 0 时阻止并列出调用。 |
| 删除定义及全部调用 | 原子批次：先删除所有调用，再删除 Graph | 放在二级危险菜单；确认框必须包含名称和调用数，不能由画布 Delete 触发。 |
| 展开调用 | 将定义内容克隆进当前父图并重连边 | 定义仍被其他调用使用时必须保留；最后一个调用可询问“保留定义”或“一并删除定义”。 |
| 复制调用 | 新增 GraphCall，仍指向同一 graphId | 两处共享后续定义修改。 |
| 复制定义 | 深拷贝 Graph、重建内部 element ID，并可让当前调用改指向副本 | 文案用“复制定义”或“创建独立副本”，不叫泛化的“复制”。 |
| 折叠 / 提取选区 | 新增 Graph 定义 + 用 GraphCall 替换选区 | 保持现有一个原子 authoring command。 |

Node-RED 选择了“删除定义同时删除所有实例”；Blender 则把 unlink 和 data-block delete 分开。Yotta 更适合以后者作为默认，因为工作流执行图具有副作用，静默级联删除调用和连线风险过高。

### 5. 端口布局必须把文字区和 handle 区分开

- 左侧端口固定“handle gutter + label”，右侧固定“label + handle gutter”；handle 不占文字流，也不压在文字末端。
- 连接点中心与每行固定高度对齐，长名称截断并提供 tooltip。
- 内部入口、出口边界与父图调用节点复用同一 `PortRow` 布局原语，避免两套 CSS 再次漂移。
- exec、error、data 依靠形状/颜色和文字共同区分，不能只依赖颜色。

## Local Application

### 当前模型已经具备的基础

- `contracts/workflow/v1/workflow-source.ts` 已明确区分 `Graph` 和 `GraphCall`。
- `EditorSession.removeGraph()` 对应删除定义；选中 GraphCall 后的普通删除对应删除实例。
- 前端 optimistic projection 和 `internal/workflow/authoring/patch.go` 都会拒绝删除仍被调用的 Graph；后端还会拒绝移除仍被 call binding/edge 使用的接口端口。
- `WorkflowGraphBoundary.vue` 已投影内部入口/出口，`WorkflowGraphCall.vue` 已按 callee Graph 投影外部端口；不需要新增 runtime 节点。
- 当前缺口主要是：图下拉菜单没有引用数和对象类型、垃圾桶语义不明显、接口只能依赖自动推断、没有展开/独立副本、错误与引用影响没有在动作前呈现。

### 不应改变

- 不把 GraphCall 注册为 Catalog node。
- 不新增 legacy subgraph runtime 或虚拟节点持久化格式。
- 不让 boundary UI 成为第二份接口状态。
- 不为了解决列表问题立即给 Graph 增加目录、标签、版本等 schema 字段；先验证搜索、名称和引用数是否足够。
- 不恢复多个执行入口；入口槽保持唯一。

### 分阶段方案

#### Stage 1 — 修正可读性与对象语义

1. 抽取共享 `GraphPortRow`，修复调用节点和 boundary 节点的 handle gutter、截断和 tooltip。
2. 把画布动作明确命名为“删除此调用”，把定义工具栏动作明确命名为“删除子图定义”。
3. 图列表展示唯一名称 + 调用数；同名时附短 graph ID。
4. 删除定义前计算引用位置：0 个引用可删除；非 0 时按钮禁用并提供“查看调用”。

验收：用户不用了解 JSON，也能回答“我删的是这个节点，还是所有地方共享的定义”。

#### Stage 2 — 完整接口编辑

1. 将现有接口面板升级为输入/输出/出口清单，支持新增、重命名、排序、删除。
2. 提供“选择内部端点”和画布拖线两种绑定方式；未绑定项显示明确状态。
3. 自动推断改名为“从未连接端口生成接口”，作为一次带变更预览的便捷操作。
4. 删除或改型前展示受影响 GraphCall；提交仍走现有 `update-graph-interface` 后端校验。

验收：用户能从空子图开始，显式创建一个入口和 `completed` / `failed` 出口，而不依赖神秘的推断按钮。

#### Stage 3 — 生命周期闭环

1. 实现“展开到当前图”，保持接口连线和节点相对位置。
2. 区分“复制调用”和“复制定义 / 创建独立副本”。
3. 增加“删除定义及 N 个调用”的原子高级动作；默认仍安全阻止。
4. 删除和展开都进入现有 undo/redo；界面明确显示“尚未保存”，说明磁盘 Source 只会在 Save 后改变。

验收：折叠是可逆的，复用和独立变体都有无歧义入口。

#### Stage 4 — 数量增长后的管理

只有真实工作流出现大量子图后，再考虑持久化 description/category/tags。第一步只做搜索、调用数、错误状态、最近打开和定位调用；避免为当前少量 workflow-local 定义引入全局资产库复杂度。

## Next Step

先实现 Stage 1，并把以下场景作为一组 UI 回归验收：

1. 折叠一个带 `completed` / `failed` 的节点，父图调用端口和内部出口标签均不与 handle 重叠。
2. 复制该调用后，管理器显示同一定义有 2 个调用。
3. 删除其中一个调用，定义和另一个调用仍存在，调用数变为 1。
4. 尝试删除定义时被阻止，并能一键定位剩余调用。
5. 删除最后一个调用后，从管理器删除定义；保存、重开后该 Graph 不再存在。

随后再进入 Stage 2；不要把“显式接口编辑”和“级联删除”混进这次纯布局修复。
