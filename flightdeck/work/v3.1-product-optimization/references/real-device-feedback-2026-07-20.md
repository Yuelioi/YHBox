# 真机反馈收集 — 2026-07-20

## Collection state

本批次正在收集，尚未形成实施 Stage。用户明确要求在全部问题补充完成前不要修改产品源码、
不要执行修复，也不要把单项反馈拆成零散任务。后续反馈继续使用 `FD-xx` 编号追加。

相关模式与一手来源见 [真机反馈 UI / 交互调研](real-device-feedback-research-2026-07-20.md)。

## Summary

| ID | 用户旅程 | 当前观察 | 已确认事实 | 等待补充或后续验证 |
| --- | --- | --- | --- | --- |
| FD-01 | 创建精准录制 | 点击录制后显示“无法开始录制”，只有错误码，流程中断 | 原图错误为 `RECORDING_CALIBRATION_REQUIRED`；当前目标未完成精准相对录制所需校准 | 校准入口是否可发现、完成校准后是否能录制、错误面是否应直接提供校准动作 |
| FD-02 | 选择视觉模板 | 模板选择器只能选择已有模板，不能在当前流程直接录制新模板 | 选择器提供搜索、筛选、最近使用和“使用此模板”，没有录制/捕获入口 | 录制后的命名、裁剪、入库和自动选中应如何保持在同一上下文 |
| FD-03 | 从节点目录添加节点 | 左侧长列表难扫描；用户希望参考 Houdini 的 Tab 一级分类与 hover 二级节点菜单 | 当前目录以搜索、分组列表和加号为主；大量相似节点仍需滚动查找 | 需比较分级菜单、模糊搜索、最近使用、键盘导航及触屏/无 hover 降级 |
| FD-04 | 运行一个简单 Workflow | Run 看起来一直卡住，用户不知道停在哪个节点 | 用户提供的 Source 含两个点击模板、一个 3 秒延迟和一个 Macro；点击模板默认 timeout 为 5000 ms，两个 timeout 分支均未连接 | 不执行 Workflow；待收集完成后查看 Run journal、当前节点、status event、timeout 分支和 Macro 终态 |
| FD-05 | 编辑区域参数 | X/Y/宽/高四项挤在一行，步进控件看不到当前值，也难理解四项关系 | `RegionValueEditor.vue` 无条件使用四列 grid；截图中字段只显露减号/加号，单位为 `%/px`，可从目标框选 | 需确定最小可读宽度、数值展示、二维区域预览和窄 Inspector 的响应式布局 |
| FD-06 | 在画布直接编辑常用参数 | 只有部分节点支持 inline 输入；现有输入字号和控件密度偏大，与 3.0 相比笨重 | 延迟节点可在展开区直接编辑 duration，但控件占据整行且信息层级弱 | 需定义哪些参数应 inline、节点/Inspector 的职责、紧凑字号与键盘输入规范 |
| FD-07 | 配置强类型 Switch | Switch 固定展示 8 个 case 输入/输出，不能按实际分支数增删 | 当前 contract 明确使用 `SwitchCaseCount = 8` 和 `typed-first-match-eight-cases/v1` | 需设计动态端口的 Source/Contract/Compiler 语义、稳定 port identity、删除分支时的连线处理 |
| FD-08 | 右键节点执行上下文动作 | 菜单没有贴近右键位置或目标节点，出现在画布远处 | 当前用 `clientX/Y - node bounds` 放置节点内隐形 anchor，再由 DropdownMenu 定位；截图显示最终菜单明显偏右下 | 需记录窗口缩放、画布 zoom/pan、DPI 与点击坐标，检查 transformed canvas 中 anchor 与 viewport collision 的坐标空间 |
| FD-09 | 创建和理解 Run 状态 | 用户不清楚它是否等同变量系统，并且找不到初始值设置 | Run 状态是每次 Run 隔离的强类型变量；Source 的 Variable 必须有 `default`，当前 UI 创建时自动选择 example/空值/0/false，但不展示或编辑初始值 | 需决定命名与说明、初始值编辑、类型变更时默认值迁移，以及与普通连线值/节点输出的关系 |
| FD-10 | 从下拉中选择窗口或其他大量选项 | 下拉直接展示所有选项；窗口达到约 20 个并含不同子类时很难扫描和定位 | 项目已有可搜索的 `USelectMenu` 用于动态目标与状态变量，但通用 `AdaptiveSelect` 和普通枚举仍使用不可搜索的 `USelect`，目前没有统一分流规则 | 调研“超过 10 项自动搜索”是否应为唯一阈值，并补充分组、元数据搜索、键盘路径、空结果与大数据集策略 |

## Evidence notes

### FD-01 — 精准录制前置条件

用户原图显示“无法开始录制 / `RECORDING_CALIBRATION_REQUIRED`”。仓库中文错误说明为“精准相对
录制需要先为当前自动化目标完成鼠标校准”。这不是录制算法失败的证据；当前可确认的问题是错误面
没有把用户带到可完成的校准旅程。

### FD-04 — 简单 Workflow 停滞

本地用户数据文件位于 `bin/data/workspace/workflows/<user-workflow-id>.json`，不复制进仓库。
只读检查确认执行链为：Run Started → 点击模板 → 点击模板 → 延迟 3000 ms → 播放 Macro。

- 两个点击模板都使用 contract 默认值：timeout 5000 ms、poll 100 ms、settle 200 ms。
- 两个节点都只把 `completed` 接到下一节点；`timeout` 与 `failed` 没有后续边。
- 因此“点击模板没有超时”与源码事实不符，但仍不能仅凭 Source 判断 UI 为什么持续显示运行中。

### FD-07 — Switch 动态分支

当前内建 contract 在 `internal/nodes/control_capabilities.go` 固定声明 8 组 `case-N` data input 与 exec
output。用户反馈不是单纯画布折叠问题，而是当前持久契约本身固定了端口集合。

### FD-05 / FD-08 — 窄面板与坐标空间

`RegionValueEditor.vue` 在所有宽度使用 `grid-cols-4`，没有基于可用宽度切换两列或单列；截图中的
数值消失与这一布局事实一致。`WorkflowNode.vue` 则把右键 client 坐标换算成 node-local anchor，
DropdownMenu 再按 viewport 做碰撞处理；在 Vue Flow 的缩放/平移 transform 中可能涉及两个坐标空间，
但在记录 zoom、pan、DPI 和点击点前不判定根因。

### FD-09 — Run 状态语义

`internal/workflow/schema/model.go` 的 Variable 属于 Workflow Source；compiler 为每次 Run 从 default
创建独立 typed state。`WorkflowStatePanel.vue` 创建变量时自动生成 default，但界面只展示名称、类型、
引用和读写动作，没有展示或编辑 default。因此“没有初始变量值”的用户感知与当前 UI 一致，即使
底层数据并非没有 default。

### FD-10 — 大量选项与分组选择

用户建议下拉选项超过 10 项时支持搜索，尤其是当前窗口约 20 个且还存在子类的场景。该建议先作为
默认触发条件候选记录，不在收集阶段直接固化成全局阈值。现有前端已经在动态目标和状态变量选择中
使用带搜索与 `> 40` 虚拟化的 `USelectMenu`，但 `AdaptiveSelect.vue` 始终渲染普通 `USelect`，节点
contract 的普通枚举也走不可搜索下拉；同类任务因此呈现不一致。后续需要同时按集合规模、是否会增长、
是否来自运行环境以及是否需要按类别/应用分组来决定控件，而不只检查当前数组长度。

## Provisional clusters

这些只是继续收集时的归类，不是实施拆分：

- **任务连续性：** FD-01、FD-02。
- **发现、密度与直接操控：** FD-03、FD-05、FD-06、FD-07、FD-08、FD-10。
- **运行可解释性与状态心智：** FD-04、FD-09。

## Prior decisions challenged by device use

- Stage I 已参考 Houdini 实现 Tab 快速添加，但验收主要证明菜单可打开、可搜索、可插入；没有证明
  用户能通过一级分类和二级节点导航发现未知节点。FD-03 说明“功能存在”不等于“可发现”。
- 现有画布边界规定一个节点只有恰好一个轻量候选时才 inline，多个 common 参数全部回到 Inspector。
  FD-06 与用户提供的 3.0 对照图说明该规则可能过度收缩直接操控，需要重新比较可读性、密度和频率，
  但在本批反馈收集结束前不改 Knowledge 结论。
- Stage I WebView smoke 证明节点尺寸、展开和插入旅程没有机械回归；FD-05/FD-08 表明它没有覆盖窄
  Inspector 数值可读性和真实 zoom/pan/DPI 下的右键菜单定位。
