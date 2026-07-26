# 真机反馈 UI / 交互调研（2026-07-20）

## Research Read

研究问题：Yotta 3.1 工作流编辑器如何降低节点发现、资源创建、参数编辑、动态分支、运行诊断与变量理解的摩擦，同时保持节点契约、类型与运行时语义可验证。

目标界面是工作流画布、节点目录、节点 Inspector、资源选择器、运行调试面板和 Run State 面板。用户任务是快速找到并创建节点，在当前上下文中捕获资源，读懂并修改参数，配置可变分支，定位一次运行卡在哪里，以及理解跨节点共享值的生命周期。

触发本次调研的是 2026-07-20 的 10 项真机反馈。本文只记录证据、模式与本仓库事实映射；不修改产品实现，不创建实施计划，也不改变 Work 的 `index.md`、`context.md` 或 `plan.md`。

约束：Windows 是完整支持平台；Yotta 的节点、类型、工作流与能力契约是 3.1 执行路径的事实来源；动态 UI 不能绕过节点契约、编译、权限或 arm 边界；鼠标交互必须保留键盘可达路径。

## Source Matrix

| 主题 | 一手来源 | 本次贡献 |
| --- | --- | --- |
| 上下文节点创建 | [SideFX Houdini Tab menu](https://www.sidefx.com/docs/houdini/basics/tabmenu.html)、[Network editor](https://www.sidefx.com/docs/houdini/network/nodes.html) | Tab 打开当前网络上下文可用节点；支持键入过滤、方向键、Enter，并可从端口或连线发起创建。 |
| 快速添加 | [Node-RED workspace nodes](https://nodered.org/docs/user-guide/editor/workspace/nodes) | Quick-Add 在鼠标处出现，混合常用、最近使用、完整列表与搜索；从连线触发时可直接插入。 |
| 分级菜单键盘语义 | [W3C APG Menu and Menubar](https://www.w3.org/WAI/ARIA/apg/patterns/menubar/) | 菜单打开后焦点、上下/左右、Enter/Space、Esc 和 Tab 的标准行为；hover 不能成为进入子菜单的唯一方式。 |
| hover 子菜单容错 | [Floating UI `useHover`](https://floating-ui.com/docs/usehover)、[`useTypeahead`](https://floating-ui.com/docs/usetypeahead) | `safePolygon`、延迟和 cursor-rest 可避免从一级移向二级时意外关闭；菜单列表可做键入定位。 |
| 设计器内录制 | [Power Automate Record desktop flows](https://learn.microsoft.com/en-us/power-automate/desktop-flows/recording-flow) | Recorder 从 flow designer 启动；可给正在开发的 flow 增补步骤；完成后动作与捕获资源回到同一设计器。 |
| 录制占位与回填 | [Power Automate Copilot recorder placeholder](https://learn.microsoft.com/en-us/power-automate/desktop-flows/copilot-in-power-automate-for-desktop) | 需要录制的流程位置可先显示占位，录制完成后原位替换；未完成时明确显示设计期缺口。 |
| 运行状态与追踪 | [Node-RED node status](https://nodered.org/docs/creating-nodes/status)、[Debug sidebar](https://nodered.org/docs/user-guide/editor/sidebar/debug)、[Power Automate run details](https://learn.microsoft.com/en-us/power-automate/desktop-flows/monitor-run-details) | 节点旁给短状态，侧栏给时间、来源与完整详情；长运行需要逐动作近实时状态、开始时间、耗时和结果。 |
| 调试控制 | [Power Automate Debug a desktop flow](https://learn.microsoft.com/en-us/power-automate/desktop-flows/debugging-flow)、[GitHub Actions Monitor workflows](https://docs.github.com/en/actions/how-tos/monitor-workflows?apiVersion=2022-11-28) | Pause、断点、单步、当前动作与实时图/日志共同回答“停在哪里”。 |
| 紧凑数值输入 | [W3C APG Spinbutton](https://www.w3.org/WAI/ARIA/apg/patterns/spinbutton/)、[WAI Labeling Controls](https://www.w3.org/WAI/tutorials/forms/labels/) | 数值控件需要直接编辑和键盘步进；每个输入需要可见且明确的标签，标签应和控件保持清晰邻接。 |
| 节点内默认值 | [Blender Node Parts](https://docs.blender.org/manual/en/5.0/interface/controls/nodes/parts.html)、[Editing Nodes](https://docs.blender.org/manual/en/5.0/interface/controls/nodes/editing.html) | 未连接 socket 可显示固定值控件；接线后由连线成为值来源；未使用 socket 和节点选项可折叠。 |
| 值来源 + 值 | [Node-RED TypedInput](https://nodered.org/docs/api/ui/typedInput/) | 紧凑控件仍同时表达值类型/来源与实际值，而不是只放无语义输入框。 |
| 动态 Switch | [Blender Menu Switch](https://docs.blender.org/manual/en/4.2/modeling/geometry_nodes/utilities/menu_switch.html)、[Unreal Add Node Pin](https://dev.epicgames.com/documentation/unreal-engine/BlueprintAPI/BlueprintEditor/AddNodePin) | 用户可增删、命名和排序条目；条目派生 socket；重命名保留已有连线；动态 pin 是节点类型能力。 |
| 右键菜单定位 | [Floating UI Virtual Elements](https://floating-ui.com/docs/virtual-elements)、[`shift`](https://floating-ui.com/docs/shift)、[`flip`](https://floating-ui.com/docs/flip) | 把 `clientX/clientY` 建模为零尺寸 viewport 锚点；用 shift/flip 处理边缘碰撞，而非手写固定偏移。 |
| 变量与初值 | [Power Automate variables](https://learn.microsoft.com/en-us/power-automate/create-variable-store-values)、[Desktop variables pane](https://learn.microsoft.com/en-us/power-automate/desktop-flows/manage-variables)、[Node-RED context](https://nodered.org/docs/user-guide/context) | 变量声明应呈现名称、类型和初始/默认值；运行时当前值与设计期默认值是不同状态；scope 与 persistence 需明确。 |
| 可搜索选择器 | [W3C APG Listbox](https://www.w3.org/WAI/ARIA/apg/patterns/listbox/)、[Combobox](https://www.w3.org/WAI/ARIA/apg/patterns/combobox/)、[Nuxt UI SelectMenu](https://ui.nuxt.com/docs/components/select-menu)、[Carbon Dropdown](https://carbondesignsystem.com/components/dropdown/usage/)、[Fluent 2 Combobox](https://fluent2.microsoft.design/components/web/react/core/combobox/usage)、[Primer SelectPanel](https://primer.style/product/components/select-panel/guidelines/) | APG 对超过 7 项的 listbox 尤其建议 type-ahead；editable combobox 可用输入过滤离散允许值。Nuxt UI 原生支持搜索字段、分组、多字段过滤、远程过滤与虚拟化；Carbon 不建议嵌套 dropdown；Fluent 将 combobox 定位于长列表；Primer 用过滤、分组、加载/错误状态处理可增长集合。 |

## Patterns

### 1. 节点发现使用“两条路”，而不是只换一种目录

**模式。** 常驻目录承担浏览与学习；画布指针处的 Tab / Quick-Add 承担已知目标的快速创建。快捷菜单先按当前图、平台、正在连接的端口类型裁剪，仍保留全文搜索与分类路径。结果可加入“最近使用/常用”，但完整、确定性的分类和搜索不能被其取代。

**什么时候用。** 常驻目录适合不知道节点准确名称的用户；Tab 搜索适合已经知道任务或节点名、正在接线或要在某处插入节点的用户。Houdini 和 Node-RED 都把快捷创建放在当前网络/鼠标上下文，而不是远离画布的全局对话框。

**分级菜单。** 一级 hover → 二级可以作为鼠标增强，但点击、右箭头或 Enter 必须同样能进入，左箭头/Esc 能逐层返回。需要 hover 时应有意图延迟或 safe polygon。二级通常足够；深层树会把“长列表难找”变成“层级难猜”。APG 规定 Tab 是退出弹出菜单，不是在菜单项间逐个移动。

**避免。** 不用纯 hover 作为唯一交互；不以分级目录替代搜索；不在所有上下文展示全部节点；不把 Tab 快捷添加做成与指针落点、当前连线无关的大型管理页。

### 2. 资源选择器应同时容纳“选择已有”和“就地创建”

**模式。** 从某个节点参数打开资源选择器时，用户的任务是“给当前参数一个可用资源”，并不局限于“从库中挑已有资源”。同一上下文应提供类型匹配的创建入口：模板对应捕获新模板，宏/输入录制对应启动相应录制。创建完成后回到原触发参数并直接成为待确认项或已绑定项。

Power Automate 官方明确允许从正在编辑的 flow 启动 Recorder，完成后把生成动作和捕获资源带回同一设计器；Recorder placeholder 进一步说明录制结果可以原位回填，而不是要求用户先离开、创建、返回、刷新、重新查找。

**状态。** 捕获/录制至少要显式区分准备、目标验证、倒计时、捕获中、暂停、处理中、可预览、失败；失败必须保留当前节点/参数上下文和可执行的恢复入口。

**避免。** 不把完整资源管理器嵌成“模态框里的另一套应用”；Apple 的 [Modality](https://developer.apple.com/design/human-interface-guidelines/modality) 指出复杂的模态层级会让用户忘记被中断任务。这里需要的是短、单路径的“创建并回填”。

### 3. “卡住”视图必须回答五个问题

运行时信息需要同时回答：

1. 当前在哪个节点、哪次 attempt；
2. 状态是 running、waiting、retrying、paused，还是已完成；
3. 正在等待什么条件/目标；
4. 已等待多久、何时超时或是否不限时；
5. 最近一次状态、输入、输出或错误是什么。

**双层呈现。** 画布节点只显示短状态（例如“等待模板 · 3.2/5.0 秒”）和明显的当前节点；运行侧栏保存时间线、耗时、输入输出、错误和可反向定位的节点链接。Node-RED 将短状态放在节点旁，把详细消息放 Debug sidebar；Power Automate 的 progressive action logging 按动作给开始、耗时和状态。

**超时不是统一保险丝。** 每个外部等待节点需要自己的超时、取消和状态心跳；工作流级最长运行时间是另一个层次。某节点的 `timeout` 分支未连接，也应在运行诊断中明确显示“已超时但无后续路由”，而不是看起来永远在转。

**避免。** 不用单个全局 spinner 代替节点级状态；不只在运行结束后生成日志；不把“没有新状态”表现成“仍正常工作”。

### 4. 紧凑参数仍需保留标签、值和单位

**模式。** 单个数值行至少保留 `短标签 + 可直接编辑的值 + 单位/类型`。空间参数作为一个对象呈现时，先显示共同语义（例如“搜索区域”），再显示 X、Y、宽、高；`%` / `px` 是整个对象的模式，不应靠用户从四个黑框猜测。

四列只有在每列仍能显示数值、标签和焦点状态时才成立。固定宽侧栏中，如果 stepper 按钮挤掉数值，应降低列数、使用两行 `X / Y` 与 `宽 / 高`、在紧凑态采用可直接输入且方向键步进的 spinbutton，或把 +/- 放到不抢占数值的交互层。WAI 要求可见标签清楚描述目的；缩小字体不能修复语义缺失。

**节点内输入。** 对未连线且最常改的 1–3 个参数，可在对应 socket 行显示默认值；连线后明确切换为“来自连线”。复杂对象和低频参数仍在 Inspector 编辑。折叠未使用项时保留“还有 N 项配置/存在错误”的信号。

**避免。** 不使用 placeholder-only；不把单位藏在上级不明显的 toggle；不把关键标签仅放 hover；不因为追求密度把全部 Inspector 表单塞进节点。

### 5. 动态 Switch 的稳定身份比“加号按钮”更重要

**模式。** 分支集合是节点实例可编辑的数据：添加、删除、命名、排序，并由其派生输入/输出端口。每个 branch 必须有独立于显示名和顺序的稳定 ID，才能让重命名/重排保留连线。Blender 明确保证 Menu Switch 条目重命名后保留对应 socket 的链接。

`default` 和 `failed` 是特殊语义分支，不应伪装成普通编号 case。删除已接线分支需要明确告知断线影响；动态 topology 也必须覆盖序列化、撤销/重做、复制粘贴、类型求解与编译，不能只在前端隐藏固定端口。

**避免。** 不用显示名作为端口身份；不通过写死一个更大的 case 上限假装动态；不静默删除已接线分支。

### 6. 上下文菜单锚定的是 viewport 点击点

**模式。** 右键菜单的 reference 是 `MouseEvent.clientX/clientY` 形成的零尺寸 virtual element。菜单可以 portal 到顶层 overlay，但需保留触发元素作为 context element，并用 shift/flip 在窗口边缘调整。打开期间滚动、缩放或布局变化时重新计算或关闭。

**避免。** 不混用画布 world 坐标、节点局部坐标、page 坐标与 viewport client 坐标；不靠固定 offset 修正 transformed/zoomed canvas 的误差；不让自动更新监听在菜单关闭后继续运行。

### 7. Run State 必须把生命周期说清楚

**推荐心智模型。** “Run State”在 Yotta 当前语义中是“每次 Run 创建并用默认值初始化、可被多个节点读写、Run 结束即结束的强类型变量”。它不是 workflow 输入，也不是跨 Run/重启自动持久化的设置。

面板中每个变量至少需要回答：名称、类型、初始值、当前值（运行时）、引用位置和生命周期。设计期初始值与运行时当前值应分栏/分态，不能共用一个含义不明的输入框。Power Automate 的变量界面同样区分 default value 与 live variable value，并允许运行暂停时观察或修改当前值。

**术语。** 单独写“Run 状态”很容易被理解为运行结果或状态机。更直白的候选表述是“运行变量”，并附固定说明“每次运行按初始值重置；仅在本次运行中跨节点共享”。若未来增加持久状态，需要单独的 scope / persistence 概念，不能静默改变现有 Run State 的可复现性。

### 8. 下拉是否可搜索由集合语义决定，数量阈值只是兜底

**判断。** “超过 10 项自动支持搜索”适合作为默认兜底，但不应成为唯一规则。APG 已经对超过 7 项的 listbox 尤其建议 type-ahead；搜索框比首字母跳转更适合窗口标题、应用名、包名这类长字符串和重复前缀。另一方面，`exact / regex`、`unique / topmost` 这类固定、短小且完整可见的契约枚举，增加搜索框只会增加操作层级。

建议按四类处理：

1. **固定小枚举：** 选项由 contract/schema 封闭定义、标签短且通常不超过 10 项时使用普通 Select；仍需完整键盘导航。若固定枚举超过 10 项，也切换为可搜索选择器。
2. **增长或环境数据：** 窗口、应用、设备、页面、资源、变量等来自用户或当前环境，数量不可预先封顶；无论当前是否达到 10 项，都采用可搜索 SelectMenu/Combobox，避免同一字段随着数据增长突然改变心智模型。
3. **分组或层级数据：** 不嵌套多个 dropdown。保留一层可见分组或路径标签，搜索同时匹配主标签、组名和辨识元数据；Carbon 明确反对 nested dropdown，Nuxt UI 和 Primer 都原生支持分组呈现。
4. **异步大集合：** 打开时再加载，搜索请求 debounce；显式展示 loading、空结果、失败和重试。数量足够大时再虚拟化或分页；虚拟化是渲染策略，不是是否提供搜索的阈值。Nuxt UI 还提示虚拟化会把分组拍平，因此不能无条件同时开启。

**窗口选择。** 20 个窗口已经明确属于可搜索集合。推荐一级按应用/进程分组，行内以窗口标题为主信息，应用/进程与窗口 class 为辅助信息；搜索覆盖标题、应用/进程、class 和完整分组路径。选中值仍绑定稳定目标身份，不允许过滤文本被误当作自由输入。保留当前选中项、键盘上下选择、Enter 确认、Esc 取消，以及无匹配/窗口已关闭状态。

**阈值表达。** 可将策略写成 `searchable = dynamicOrGrowing || hierarchical || itemCount > 10`；`virtualize` 另由实测规模决定。这样用户提出的 10 项阈值得到执行，同时不会让 8 个动态窗口使用普通下拉；12 个固定枚举虽然获得本地过滤，也不会因此被误做成远程搜索。

## Local Application

| # | 用户反馈 | 本仓库事实 | 研究判断（仅记录，不执行） |
| --- | --- | --- | --- |
| 1 | 精准录制失败 | `frontend/src/i18n/zh.ts:1314` 已有 `RECORDING_CALIBRATION_REQUIRED`，含义是当前自动化目标尚未完成精准相对录制所需的鼠标校准；`frontend/src/composables/useRecordingStart.ts:23` 在打开 HUD 前先校验目标。 | 这是有明确前置条件的失败，不应只显示通用“无法开始录制”。录制入口需要在开始前展示校准状态，并给出保持当前上下文的校准/切换录制模式恢复路径。 |
| 2 | 从节点的“选择模板/资源”流程无法直接录制或捕获 | `frontend/src/components/assets/AssetPickerModal.vue:16-231` 只提供搜索、筛选、已有项选择与确认，没有创建入口；`frontend/src/app/editor/WorkflowResourceDock.vue:37-60` 其实已经按 tab 提供录制宏、精准录制和捕获模板。 | 能力已经在编辑器资源 Dock 中存在，但节点参数的选择器没有同样入口，形成上下文断点。选择器应按资源类型复用“创建并回填”能力，而不是要求用户关闭选择器后去另一个面板。 |
| 3 | 节点目录难找，希望 Houdini 式 Tab + 一级/二级目录 | `frontend/src/views/WorkflowEditorView.vue:2599-2608` 已在鼠标位于画布时拦截 Tab；`frontend/src/app/editor/WorkflowQuickAddMenu.vue:1-84` 已有分类、搜索、上下键和 Enter，但以 `BaseModal` 的 30rem 双栏呈现；左侧常驻目录位于 `WorkflowEditorView.vue:250-320`。 | 需求不是“从零加 Tab”，而是让现有 Quick-Add 更指针局部、上下文相关且可浏览。保留常驻目录；Tab 菜单支持搜索 + 至多二级分类 + 最近使用/兼容节点。hover 只能增强，方向键/点击必须等价。 |
| 4 | 简单工作流运行后一直卡住，不知道停在哪里 | 用户本地文件 `bin/data/workspace/workflows/<user-workflow-id>.json` 含两个串联 `click-template`，两者都只连接 `completed`，未连接 `timeout` / `failed`；`internal/nodes/automation_template.go:121-125` 默认 timeout=5000ms、poll=100ms、settle=200ms；`internal/noderuntime/automation_template.go:75-87` 未命中会选择 `timeout`。`internal/workflow/compiler/scheduler.go:215-239` 在 adapter 运行前记录 attempt started，`frontend/src/app/editor/runTrace.ts:3-29` 能映射为 running。 | 现有底层有节点开始事实与 5 秒节点超时，但用户仍无法从主画布得知当前等待节点、已等待时间、最后匹配结果与未连接分支。不能仅推断是模板超时；也可能已经走到后续宏或外部调用。应先把“当前节点/等待原因/倒计时/分支结果”可视化，才能可靠诊断。 |
| 5 | 区域的 X/Y/宽/高挤在一起，看不清值和含义 | `frontend/src/app/editor/RegionValueEditor.vue:27-38` 在 360px Inspector 内用四列 `UInputNumber`；每列带 label，但 stepper 按钮会占用窄输入宽度；单位只在上方 `%/px` 切换。 | 这是密度与可辨识度问题，不是缺少数据。四列应改为在当前宽度仍能读值的排布，并在标签/单位上维持对象关系；不能靠继续缩字体解决。 |
| 6 | 有些节点支持直接输入，但字体大；3.0 支持更多且更易读 | `frontend/src/app/editor/WorkflowNode.vue:155-185` 具备节点内 inline input；节点固定宽 260px（`:3`）。`frontend/src/app/editor/authoringSurface.ts:64-71,119-149` 只允许一组紧凑 adapter，且当 inline 候选不恰好等于 1 时全部清空。 | 当前不是“不支持行内参数”，而是投影规则刻意只显示恰好一个候选，导致许多节点没有行内值。可读性目标应是 socket 行的标签/值/单位与连接态，而不是简单恢复“越多越好”；复杂对象继续留在 Inspector。 |
| 7 | Switch 不应写死输入输出，应可直接设置 | `internal/nodes/control_capabilities.go:21,32-39,57-60` 固定 `SwitchCaseCount=8`，conformance 是 `typed-first-match-eight-cases/v1`；前端只是忠实投影固定 contract。 | 这是契约与序列化能力缺口，不是单纯节点 UI 缺少 +/-。方向应是实例级动态 branch schema + 稳定 branch ID + 类型一致性；default/failed 保持特殊端口。 |
| 8 | 节点右键菜单位置不对 | `frontend/src/app/editor/WorkflowNode.vue:354-362` 将点击点转成 `event.clientX - node.left` / `event.clientY - node.top` 的节点局部坐标，再用节点内隐藏按钮作为 dropdown anchor；画布本身存在缩放/变换。 | 截图现象与坐标域混用高度一致。成熟做法是直接以 viewport client point 作为 virtual reference，再做碰撞 shift/flip；不应继续在节点局部 offset 上打补丁。 |
| 9 | Run State 像变量系统但没有初始值 | `contracts/workflow/3.1/workflow-source.ts:181-185` 的 Variable 要求 `default`；`internal/workflow/compiler/state.go:70-88` 在每次 Run 创建 state 并从 initial/default 初始化；`frontend/src/app/editor/WorkflowStatePanel.vue:311-345` 创建变量时自动选择 example、空串、0、false、首个 enum 等默认值，但 `:36-58` 创建 UI 只有名称与类型，`:68-205` 列表也不显示/编辑 default。 | 用户理解正确：它本质上是 per-run 变量系统，但 UI 隐藏了 schema 已要求的初值，导致术语和行为脱节。应显式呈现/编辑初始值，并说明“每次运行重置、运行内跨节点共享、非跨运行持久化”。 |
| 10 | 选项多或有子类时，下拉直接列全量内容难找；建议超过 10 项支持搜索 | `frontend/src/components/common/AdaptiveSelect.vue:1-43` 始终包装普通 `USelect`，没有根据集合语义/数量切换搜索；`frontend/src/app/editor/GeneratedFieldEditor.vue:3-40` 的目标资源和状态变量已使用可搜索 `USelectMenu`，40 项以上虚拟化，但 schema `select` 枚举仍固定使用 `USelect`。`frontend/src/views/SettingsAutomation.vue:864-922` 的已安装应用、ADB 设备、浏览器页面都由环境数据生成，其中应用、设备和页面仍走 `AdaptiveSelect`，只有 Android 应用在 `:376-389` 已有搜索与 40 项以上虚拟化。桌面窗口配置当前不是实时列举全部窗口，而是由捕获结果回填应用 slot、标题和 class（`:1479-1603`）；`windowTitleMatch` 与 `windowSelection` 各只有两个 schema 选项。 | 这是选择器策略不一致，而非所有下拉都应改成搜索。采用 `dynamicOrGrowing || hierarchical || itemCount > 10` 作为搜索规则；固定小枚举继续普通 Select。未来/现有窗口类选择必须可搜索并按应用/进程分组，匹配标题、应用/进程、class；异步加载、错误/空状态、稳定 identity 与键盘语义一并纳入。现有 `>40` 只适合作为虚拟化策略，不应被误用成搜索阈值。 |

### 跨项原则

1. **上下文优先。** 添加节点、创建资源、查看日志都应保留触发画布位置、节点和端口。
2. **语义优先于密度。** 缩小字体或隐藏标签不能替代清楚的名称、值、单位、来源与生命周期。
3. **短状态在画布，完整证据在侧栏。** 两者通过节点 ID / attempt 相互定位。
4. **动态拓扑从 contract 开始。** Switch 的 UI、schema、撤销、边、类型求解和编译必须共享稳定身份。
5. **鼠标增强不得破坏键盘路径。** hover 子菜单、右键菜单和 spinbutton 都要遵守标准焦点与键盘语义。
6. **默认值必须可见。** 未连接端口默认值和 Run State 初始值都会实际影响运行，不能只存在于 schema。
7. **失败要附恢复动作。** 校准缺失、超时未接、目标不可用都应告诉用户下一步可做什么，同时保留当前编辑上下文。
8. **搜索看集合语义，虚拟化看规模。** 动态/层级集合默认可搜索，10 项只是普通 Select 的数量上限兜底；虚拟化、分页或远程过滤单独决定。

## Next Step

按用户要求暂不执行。本文作为后续反馈汇总的证据基线；在用户明确“全部说完”之前，只继续追加问题与事实，不据此修改源码、节点契约或 Work 计划。
