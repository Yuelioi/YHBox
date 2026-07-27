# V4 goal context

## Product truth

- Yotta 是为个人用户和开发者设计的本地自动化工作台，不是企业发布、审批或软件供应链平台。
- V4 是后续产品体验、功能和稳定性优化的唯一持续路线；3.1 Work 只保留为历史，不再恢复其阶段计划。
- Workflow 是唯一一级内容对象。Target 只在工作流需要连接本机对象或服务时出现。
- Schedule 是可选的自动触发方式；Resource 和 Run 是围绕 Workflow 的支持信息，不要求用户先学习。
- 高级能力必须存在，但通过上下文和渐进披露出现，不能与打开、编辑、运行争夺首屏。
- 编辑器窄栏有足够纵向空间时直接平铺不同工作区入口，不为同层少量工具增加资源二级菜单。
- 编辑器只保留一层命令顶栏：左侧展示 Workflow / 图路径，右侧放撤销、重做、定位、Target、运行、
  保存和工具；Target 紧邻运行并保持当前值可见。
- 单层顶栏不能依赖横向滚动隐藏命令：左侧 Workflow / 图路径是可截断区，右侧核心动作不可收缩；
  窄窗口可折叠 Target 文字，但必须保留图标、可访问名称和提示。
- 画布左上使用“添加节点 / 更多”分裂入口，注释和子图调用作为同组次级创建动作；显式添加节点从
  触发源附近展开，Tab 快速添加继续跟随当前画布光标位置。
- 只适用于子图接口的推导动作由子图接口面板持有，不进入全局编辑器顶栏。
- 菜单项图标和文字统一左对齐，不能用居中排版削弱扫描路径。

## Experience direction

- 设计语言是冷静、直接、工具感强；继续使用现有 Nuxt UI、深色中性色和单一青绿色强调色。
- 默认页面只服务高频动作；筛选、批量管理、导入导出、恢复等低频动作进入明确的管理入口。
- 正常运行不弹授权或安装流程。缺少 Target、Credential 或 Node 时，直接说明缺什么以及如何修复。
- 动效只用于状态反馈和空间关系，不用于装饰。

## Stability constraints

- 保留现有节点编辑、子图、Snippet、Macro、InputClip、录制、视觉模板、资源、Target、Schedule、
  调试、日志、时间线和运行历史能力。
- `fishing-v2` 是 V4 的真实黄金工作流；每个影响 Source、节点契约、资源或运行路径的阶段都必须验证。
- 每用户应用路径、窗口和设备配置属于 Settings；Workflow 不保存这些路径，运行时通过可回滚的
  automation generation 原子热更新。
- Windows WebView 的默认启动预算为 15 秒，Workflow 首屏预算为 5 秒；黄金数据验收必须复制到
  隔离 profile，不能写入原始用户数据。
- V3 恢复点 `645e0bad` 和 V4 技术基线 `e330f47b` 都必须可恢复。
- 持久化合同独立演进；迁移必须可验证、可恢复，失败时不得发布部分结果或丢弃旧数据。
- 不建立永久 `internal/v4`、第二套 runtime 或只有一个实现的预留 interface。
- Go 清扫按决策归属和 Module 深度推进，不以文件行数、目录搬迁或 interface 数量作为完成证据。
- source compatibility 与 durable compatibility 分开处理：无生产调用的别名可直接删除；已发布的
  settings、node package、Run 与 Blob 读取路径只有在完成持久化改写和版本退役证明后才能移除。
- Workflow Source 和 Program 都使用 canonical JSON，但含义不同：Source 是可编辑文档，Program 是
  展开子图、解析类型和端口、锁定实现、排序执行并封存 capability plan 后的执行计划。内部可继续把
  这段转换称为 compile；产品界面只称“检查工作流”，不让用户误以为会生成二进制或可部署成品。

## Definition of done

- 用户启动应用后无需配置即可浏览已有 Workflow 或新建 Workflow。
- 已有 Workflow 一次点击打开，一次点击运行；正常路径不经过 Modal。
- 默认工作流首页只显示一个主操作、一个搜索入口和工作流内容，不常驻企业式管理工具栏。
- 编辑器默认突出画布、保存状态和运行；调试、AI、资源管理在需要时出现。
- 完整功能保留基线、真实数据迁移、`task check`、Windows build 和 WebView 主旅程全部通过。
