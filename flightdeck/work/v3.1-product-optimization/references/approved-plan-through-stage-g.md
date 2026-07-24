# 3.1 产品优化总方案

## Outcome

3.1 编辑器在不恢复旧 runtime 的前提下，具备专业画布、子图、录制、调试、节点发现、资源编辑
和管理创作闭环，并通过统一门禁与真实 Windows WebView 旅程。

## Current stage

Stage A–I 已完成，当前没有活动实施 Stage。只有收到新的真机反馈或明确产品范围后才追加下一
Stage；已验收的 Stage I 不再拆回零散修补。

下方主体保留 Stage A–G 的详细方案；Stage H 的节点上下文菜单与模板创作、Stage I 的紧凑节点、
资源编辑、Tab/Snippet 和计划 Modal 证据保存在 `references/13-*` 至 `references/17-*`。

## 1. 为什么单独开 Topic

此前旧能力路线的职责是“从旧系统迁移并补齐到 3.1”，其中混合了架构迁移、能力恢复、真机修复和大量历史结论。继续追加产品体验优化会产生三个问题：

1. 已完成的迁移结论与新的真机否定结论互相覆盖。
2. “底层能力存在”容易继续被误写成“用户已经可用”。
3. Topic 过大后，恢复上下文、阶段验收和责任边界都不清楚。

因此 `v3.1-product-optimization` 只负责 3.1 的产品优化与可靠性收口；已完成的升级历史由 Git 保存，不再作为当前搜索范围中的 Work 文件保留。

## 2. 产品设计定位

Design Read：这是面向 Windows 自动化创作者和高频专业用户的桌面 IDE，不是营销页，也不是低密度表单应用。延续现有暗色、单一绿色强调色、Nuxt UI 和 Tabler 图标体系，采用定向演进而不是视觉推倒重来。

设计参数：

- 视觉变化度：3/10。稳定、可预测，避免为了“新”改变已有肌肉记忆。
- 动效强度：2/10。只用来表达状态变化，不做装饰动画。
- 信息密度：8/10。允许高密度，但必须通过分组、渐进披露和任务控件降低认知负担。
- 工作区结构：保留专业工具常见的左侧目录、中央画布、右侧检查器、底部可折叠运行面板，不引入完整自由停靠系统。

## 3. 架构判断

当前 3.1 的核心框架没有需要推翻的问题。健康的部分包括：

- Workflow Source、Compiler、Program、Admission 和唯一 Scheduler 仍是正确主干。
- GraphCall 不是 Catalog 节点，也没有第二套 runtime，子图会编译进同一 Program。
- Node Contract 与 Data Type 是运行语义来源，Authoring Projection 是正确的创作扩展点。
- DebugController、Run timeline 和日志事实都有真实后端接口，不是纯前端假界面。

真正的问题位于产品表达层和验收方式：

- 状态没有分层，同一个边框同时承担选中和运行结果。
- 能力入口存在，但缺少直接手势、边界投影和任务导向控件。
- 通用 schema 表单被当作最终体验，复杂类型虽然严格，却仍要求用户理解底层数值。
- 单元测试和静态页面曾被用来代替真实工作流闭环，导致“实现完成”与“真机不可用”并存。

结论：不回滚 3.1，不恢复旧 Container runtime，不复制旧按 kind 分发。需要深化 Authoring 模块和编辑器交互契约。

## 4. 六个问题的处理决策

| 问题 | 已确认事实 | 方案决策 |
| --- | --- | --- |
| 运行后所有节点绿色边框，选中无反馈 | WorkflowNode 同时用 border-primary 表示选中、用 border-success 表示成功，后声明的运行状态覆盖选中态；节点头部已经有运行状态标签 | 边框或外环只表示选择与键盘焦点；运行状态改用头部状态标识、节点侧边短条和执行连线轨迹；调试和校验再使用独立通道 |
| 框选消失 | 批量工具栏、删除、复制、布局、折叠子图命令仍在；Vue Flow 当前只禁用了默认 Delete，没有显式 selection-on-drag、selection key 或平移契约 | 建立明确的画布鼠标契约：空白处左拖框选、Shift 追加、Ctrl 切换、Esc 清空、Delete 删除、Space 或中键拖动平移；把手势帮助放到可发现入口 |
| 子图内部没有 Start/End，流程不可理解 | Graph 有 typed inputs/outputs、entries 和命名 exits；GraphCall 只有一个 in；collapse 明确拒绝多个不同执行入口，但允许多个 exec/error 出口 | 在子图画布投影一个不可删除的“子图入口”和每个命名出口对应的“子图出口”，它们只属于 authoring，不进入 Catalog/runtime；调用节点镜像接口；真正多入口不在本次优化中暗改 |
| 调试单步第三次卡住 | 后端有真实 DebugController 和 scheduler checkpoint，前端也有 Start/Step API；但真实三节点工作流仍无法闭环 | 设置保留或下线硬闸门。只有 Start、Step、Continue、Stop、失败、循环和 GraphCall 真机闭环全部通过才保留产品入口；否则 3.1 隐藏调试、断点和误导文案，只保留日志与时间线 |
| 复杂节点全是底层数值 | Authoring Projection 只有基本 control、constraints 和 editorAdapter，没有分组、单位、主次、预设；ColorRange 编辑器只显示 RGB/HSV min/max；ScreenPicker 已有 ExtractColorRange 能力但未形成编辑闭环 | 扩展 authoring presentation 元数据，并建立类型级控件注册表。默认提供任务控件、采样、预览和人类标签；底层通道值进入“高级”区，不删除严格类型 |
| 简易录制与精准录制混在同一个模式选择中，简易宏丢失并发按键语义 | InputClip 底层已经是 KeyDown、KeyUp、MouseDown、MouseUp、Move、RawDelta、Scroll 原子事件；损失发生在上层把一段重叠按键压成 keys[] + duration，再重新串行展开 | 在产品领域、入口、资源库、编辑器和回放节点上拆成“宏”和“精准录制”两套系统；简易宏使用显式原子动作和 Sleep，精准录制保留原始时间流与鼠标轨迹 |

## 5. 编辑器状态系统

四类状态不得再竞争同一视觉通道：

| 状态 | 唯一主通道 | 辅助表达 |
| --- | --- | --- |
| Selection / Focus | 稳定的 2px 绿色外环或 outline | 多选数量工具栏、键盘焦点 ring |
| Execution | 节点头部状态文字与语义色短条 | 当前执行连线、可清除的最近一次运行轨迹 |
| Debug | 琥珀色当前暂停标记和断点槽 | Debug 面板中的“下一步将执行”说明 |
| Validation | 端口或字段就地错误、节点错误徽标 | 问题面板定位，不使用全节点选中边框 |

运行结束后仍允许选择、拖动和编辑节点。运行轨迹可以保留，但必须支持清除，并且不能阻断编辑器命中测试。

## 6. 子图产品模型

3.1 本次采用清晰且与现有编译契约一致的模型：

- 父图中的 GraphCall 有一个执行输入 in。
- 子图内部显示一个不可删除的“子图入口”，连接到真实首节点。
- 子图可以有多个命名执行或错误出口，每个出口在画布上显示独立“子图出口”。
- typed data inputs/outputs 由接口面板管理，并在入口、出口或调用节点上以端口形式呈现。
- 双击调用节点进入子图，顶部面包屑返回父图。
- “折叠为子图”自动推导数据边界、一个执行入口和多个出口；发现多个不同执行入口时就地解释为什么不能折叠，并定位冲突边。
- 子图调用节点、接口面板和内部边界投影来自同一份 Graph interface，不允许三份状态各自修改。

这不是恢复旧版独立子图 runtime。旧版的 virtual markers 只作为可视化经验被吸收，保存和执行仍使用 3.1 Source-native GraphCall。

## 7. 复杂节点创作模型

采用三层深模块，不让页面按节点 ID 堆条件分支：

1. 通用投影底座
   负责标题、说明、端口、默认值、required、类型校验、capability、可用平台和基础控件。

2. 类型级 Editor Adapter
   按数据类型或 editorAdapter 注册交互。首批统一 Duration、Point、Region、ColorRange、KeyChord、Asset、Target。控件只编辑已声明的 typed value，不拥有运行语义。

3. 任务配方与预设
   为常见目标提供可搜索的组合入口，例如“检测某种颜色是否出现”“查找色块”“等待模板出现”。配方生成普通 3.1 节点或 Source-native 子图，不创建第二套节点系统。

Analyze Color 的目标体验：

- 先选择图像来源和搜索区域。
- 支持从已绑定目标截图中取样颜色，复用现有 ScreenPicker 与 ExtractColorRange。
- 默认显示色样、容差或范围、区域和匹配预览，不要求用户先理解 H/S/V 通道上下限。
- RGB/HSV 通道最小值和最大值保留在高级区。
- 输出使用“匹配像素数、覆盖比例、中心位置”等用户语言，并给出下一步可连接建议。
- Inspector 按“必填、常用、高级、输出”分组；节点卡片最多内联 1-3 个高频且未连线的输入。

Authoring Projection 需要补充但不改变 runtime 的 presentation facts：group、order、importance、unit、inlinePriority、preset、help 和 editorAdapter。所有展示信息由一个注册表消费。

## 8. 录制产品模型

简易录制和精准录制不是同一种资源的两个显示模式，而是两个不同用户任务。底层可以复用 Windows hook、事件编码和安全释放机制，但产品领域必须分开。

### 8.1 简易录制：可编辑宏

独立入口命名为“录制宏”，并支持“新建空白宏”。宏资源使用版本化的有序 MacroAction 文档，不再把动作表示为 keys[] + duration + 隐藏 delay。

首批动作语法：

- KeyDown(key)
- KeyUp(key)
- MouseDown(button, point)
- MouseUp(button, point)
- Click(button, point)
- Scroll(point, notches)
- Sleep(duration)

Click 是方便普通点击的高层原子动作，回放时确定性展开为 MouseDown、短 Sleep、MouseUp。需要跨动作保持鼠标按住状态时使用 MouseDown 和 MouseUp。连续鼠标移动、拖拽轨迹和视角转向不进入简易宏，它们属于精准录制。

W 和 D 交叠的正确表达示例：

1. KeyDown(W)
2. Sleep(50ms)
3. KeyDown(D)
4. Sleep(450ms)
5. KeyUp(W)
6. Sleep(100ms)
7. KeyUp(D)

不能把它压成“W + D，持续 600ms”，因为这会丢失每个键独立的按下和释放时刻。

宏编辑器要求：

- 一行只表达一个动作，不把延迟藏在上一行或下一行字段里。
- 支持录制、手动插入、删除、复制、批量选择、拖拽排序和动作搜索。
- KeyDown/KeyUp 使用按键捕获器，不要求手输字符串。
- 选择某一行时展示该时刻的“当前已按住按键和鼠标按钮”。
- 重复 KeyDown、无对应 KeyDown 的 KeyUp、结尾未释放输入必须就地提示。
- 停止、取消、失败和应用退出都必须释放所有 held input。
- 保存后仍是宏资源，不自动“添加到画布”；用户在工作流中显式选择“回放宏”节点并绑定资源。
- 不把宏自动拆成 PressKeys、Delay 等线性节点，因为跨节点持有 W 再按 D 的语义会再次丢失。

### 8.2 精准录制：不可随意压缩的输入轨迹

独立入口命名为“精准录制”，使用独立资源类型和“回放精准录制”节点。它保留：

- KeyDown、KeyUp、MouseDown、MouseUp 的原始顺序。
- 鼠标绝对移动、相对 RawDelta、拖拽、滚轮和微秒时间戳。
- 录制分辨率、mouse mode、counts360 和目标环境快照。
- 同一时间戳下的 Seq 顺序。

精准录制不进入宏动作列表，也不自动压缩成 Click、组合键或 Sleep。它的编辑器是轨迹和时间线工作台，首批只提供不会破坏语义的操作：

- 播放预览与事件统计。
- 起点和终点裁剪。
- 暂停区间处理。
- 轨迹、按键和鼠标按钮分轨查看。
- 分辨率与 counts360 校准提示。
- 明确的原始事件查看入口，仅供高级用户诊断。

后续若增加平滑、速度缩放或片段编辑，必须作为精准录制自己的能力验收，不能复用宏编辑器的行操作。

### 8.3 产品隔离规则

- 资源库分别提供“宏”和“精准录制”入口、列表、筛选、图标和详情，不使用模式下拉框切换。
- 工作流编辑器分别提供“回放宏”和“回放精准录制”节点，端口类型不得互换。
- 录制开始前直接选择任务入口，不在一个录制按钮后再选择 simple/precise。
- 两者不能隐式互转。若未来提供“从精准录制提取宏”，必须是显式、有损、可预览的命令。
- 共享的 hook、event codec、回放 backend 和 held-input 清理属于内部实现，不得让 UI 看起来仍是一种资源。
- 两种录制都遵守先保存资源、再由工作流显式绑定的流程，不自动修改当前画布。

## 9. 分阶段实施计划

### 阶段 A：画布心智模型

包含 Slice 02 和 Slice 03。

Slice 02：状态通道与专业级选择

- 分离 selection、execution、debug、validation。
- 恢复框选、追加选择、切换选择、快捷删除和平移手势。
- 保持批量复制、删除、对齐、分布、自动布局和折叠子图的直接入口。
- 增加运行轨迹清除和画布手势帮助。

Slice 03：Source-native 子图创作

- 增加子图入口与命名出口的 authoring 投影。
- 接口面板、调用节点、内部边界和面包屑使用同一 Source 事实。
- 完成折叠选择后的自动重连、错误解释和跨图定位。
- 不引入多入口语义，不恢复旧 runtime。

阶段验收一次执行：

- WebView 中完成框选、Shift 追加、Delete、对齐、折叠子图、进入与返回。
- 运行完成节点仍能清楚显示选中态。
- 一个入口、完成出口和失败出口的子图可读、可保存、可运行。
- 运行阶段聚合前端测试和编译相关定向测试，阶段末再执行 task check 与真实 Windows WebView smoke。

### 阶段 B：宏与精准录制分轨

包含 Slice 04 和 Slice 05。

Slice 04：原子宏模型与宏编辑器

- 将简易录制从 grouped keys 动作改为 KeyDown、KeyUp、MouseDown、MouseUp、Click、Scroll、Sleep 的 tagged union。
- 建立 Macro asset、编辑器、验证器、held-input 状态机和“回放宏”节点。
- 支持录制、手动创建、增删、复制、批量选择、排序和按键捕获。
- 停止、取消、失败和退出统一释放所有 held input。
- 保存资源不自动修改画布。

Slice 05：精准录制工作台

- 保留完整 InputClip 原始事件、轨迹、RawDelta、拖拽、滚轮和微秒时间。
- 建立独立录制入口、资源类型、预览工作台和“回放精准录制”节点。
- 首批提供播放预览、起止裁剪、暂停区间处理、事件分轨查看和校准提示。
- 不经过宏动作压缩，不与 Macro asset 隐式互转。

阶段验收一次执行：

- 真机录制 W Down、D Down、W Up、D Up，编辑、保存、重开和回放后顺序与交叠保持不变。
- 手动创建 KeyDown、Sleep、KeyUp、Click 宏并成功回放。
- 任意取消或失败路径结束后不存在残留按键或鼠标按钮。
- 精准录制能保留并回放鼠标转向、拖拽、滚轮和交叠按键。
- 产品中不再出现 simple/precise 模式下拉框，宏和精准录制拥有独立入口与资源列表。
- 聚合录制 contract、codec、回放和真机测试后，再执行阶段完整门禁。

### 阶段 C：调试器去留闭环

包含 Slice 06。

Slice 06：真实调试链路和运行工作台

- 以用户失败的三节点工作流作为第一真机 fixture。
- 从点击“调试”到 Application、worker、scheduler checkpoint、事件回传和前端 generation 合并逐层记录事实，定位卡住位置。
- 定义 starting、paused、stepping、running、terminal 的单一状态机和控制确认语义。
- 普通运行不自动展开调试面板；只有暂停或用户主动打开时显示。
- 日志、时间线、调试是同一底部工作区的独立页签，各自回答不同问题。

保留闸门：

- Start 必须停在首个即将执行节点之前。
- Step 每次只执行一个节点，effect 恰好一次。
- Continue 能完成，Stop 不遗留 paused Run。
- 失败路由、Repeat/ForEach/Retry 和 GraphCall 至少各有一条可信路径。
- 连续多次启动与单步不能复用旧 runId 或旧 generation。

任一关键条件无法在真实 WebView 和 UAC 构建中闭环，则从 3.1 产品 UI 移除调试、断点和单步入口。内部代码是否保留另行决定，但不能继续向用户展示半成品。

### 阶段 D：节点创作体验

包含 Slice 07 和 Slice 08。

Slice 07：Authoring Surface 深模块

- 扩展 presentation metadata。
- 建立类型级 Editor Adapter 注册表。
- 统一节点内联输入、Inspector 分组、单位、帮助、默认值和高级折叠。
- 统一 Region、Point、Duration、KeyChord、Asset、Target 的使用模式。

Slice 08：视觉分析与配方闭环

- 重做 ColorRange，…743 tokens truncated…真正多入口另开架构决策。
2. 调试采用真机硬闸门；通过则保留，未通过则从 3.1 产品入口下线。
3. 复杂节点采用通用投影底座 + 类型级 Editor Adapter + 任务配方，禁止旧 runtime 和节点 ID 条件树。
4. 宏与精准录制从领域、入口、资源、编辑器和回放节点分轨。

用户已批准总方案。实施从阶段 A 开始，每个阶段完成后集中验收和提交；中间 Slice 只做继续开发所需的最小定向检查。

### 阶段 F：画布节点密度与缩放恢复

包含 Slice 11。

- 复杂 Editor Adapter 只在 Inspector 或明确弹层承载完整表单；画布节点卡片只显示端口、状态和最多 1–3 个高频摘要/输入。
- 固定画布 wheel ownership：鼠标位于节点卡片时仍缩放画布，只有明确的弹层或独立滚动工作区可以消费滚轮。
- 以“分析颜色”作为高复杂度回归 fixture，锁定合理节点尺寸、命中测试、连接与缩放行为。
- 定向前端测试与真实 WebView 视觉检查通过后，在阶段末统一执行聚合门禁并提交。
- 结果：画布仅内联轻量 editor adapter，复合 typed editor 留在 Inspector；节点上的滚轮按光标锚点缩放画布。Analyze Color 真机截图、受信任滚轮输入、`task check`、编辑器 smoke 和 production build 已通过。

### 阶段 G：完整画布缩放与 Snippets 恢复

包含 Slice 11（恢复）和 Slice 12。

- Slice 11：在无预选节点的前提下，空白画布、未选中节点和已选中节点上的滚轮均由同一画布相机契约处理；下拉、弹层和明确独立滚动区除外。
- Slice 12：删除硬编码“常用配方”入口与实现；按 3.0 的用户自定义 Snippets 心智模型，在 3.1 Source/Node Contract 上恢复带配置节点模板、元数据、搜索、管理和插入画布闭环。
- 3.0 的 localStorage、Container runtime 和旧 kind 分发只作为行为取证，不直接复制；3.1 使用 durable application service 和精确 NodeRef。
- 阶段完成后统一运行聚合测试、`task check`、真实 WebView smoke 与 production build。
- 结果：空白画布、未选中节点和已选中节点上的滚轮缩放使用同一相机契约；硬编码 recipes 已移除，Snippets 以 application service、精确节点快照、元数据管理、点击/拖放插入和真实 WebView 用户旅程完成闭环。
