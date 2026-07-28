# V4 后续稳定性收尾上下文

## 已确认事实

- 作者补丁执行器使用 `$<handle>` 引用同一批次刚创建、尚未分配持久化 ID 的节点。
- `set-config`、`connect`、`disconnect` 等作者命令允许该语义；补丁专用 `PatchNodeReference`、
  `PatchEndpoint` 与 `PatchEdge` 负责表达正式 ID 或 `$handle`，持久化 Source 继续使用严格
  `schema.Endpoint`。
- `启动已安装应用` 节点只保存设置中的逻辑应用槽位，例如 `htgame`；可执行文件与固定参数继续由设置
  档案拥有，工作流不应复制路径或命令行。
- 直接通过 Application/Workflow service 保存启动节点、`htgame` 槽位和相同 exec 连线可以成功，
  因此不要修改应用槽位规则或运行时启动适配器来规避本问题。

## 修复边界

- 使用作者补丁专用的节点引用/endpoint/edge contract 表达正式 ID 或 `$handle`。
- 不放宽持久化 Workflow Source 的 Endpoint grammar。
- 重新生成并检查 tracked Workflow authoring contract；不要手改 `frontend/bindings/`。
- 回归必须覆盖新增节点、设置配置、连接信号在同一 PatchRequest 中通过生成 schema 与执行器。
- 开发阶段只跑定向测试；全部修改完成后只运行一次 `task check`，不默认运行 `task check:full`。
- 工作流保存横幅必须区分真正的 revision conflict 与普通保存失败；内部错误码不能单独作为用户正文，
  已知错误给出原因和处理建议，未知错误说明草稿仍保留并把技术码降为辅助信息。
- `targetDefaults[].slot` 是 Host Profile 中的本机逻辑槽位；它可以与可选的便携
  `targetProfileDefinitions` 同名以辅助安装，也可以只引用本机已有槽位。不得强制每个默认槽位都内嵌
  一份便携目标定义，也不得把工作流默认值复制进每个节点配置。
- 同一个异常只能有一个主要反馈表面：编辑器保存失败由 `saveError` 持久横幅独占，初次打开失败由
  `openFailure` 独占；关闭一种反馈后不得从另一份状态重新露出同一错误。
- 作者补丁错误通过 RPC 信封携带稳定 code 与 `commandIndex`；能映射到节点命令时提供“定位问题”，
  不能定位时仍须给出可执行的修复建议。
- 业务组件不得直接使用 `UModal`，统一通过 `BaseModal` 管理生命周期；详细选择规则见
  `flightdeck/knowledge/frontend/feedback-surfaces.md`。

## 节点显示规则

- 画布、检查器和作者面板使用同一个端口标题解析：节点专属 `titleKey` 优先，否则使用公共语义标签。
- 控制信号也使用公共语义标签；稳定的端口/信号 ID 只作为技术身份，并在画布悬浮提示中保留。
- 当前内置节点的无专属标题端口和全部信号必须同时具备中英文公共标签；新增内置 ID 由测试强制补齐。
- 节点内联值编辑器使用紧凑尺寸，完整检查器保持标准尺寸。
- 检查器字段的完整说明必须使用 `UFormField.description` 下置显示；`hint` 只用于极短同行状态。
  参数名使用 12px 主标签、说明使用 11px 次级文字，必填由标准 `required` 星号表达。
- 未连接且未绑定的可选泛型输入可以在 Program 中保持未实例化；必填、已绑定或已接线端口仍必须解析
  为具体类型。Log 节点依靠这条规则支持 config 手填消息，接线存在时由接线值覆盖。
- 同版本节点契约迁移只要求已有配置、绑定与拓扑在当前 Projection 中仍可表示；节点是否已填写全部
  必填值属于编译诊断，不得阻止摘要升级并制造派生的未知端口错误。编辑器检查和运行前刷新 Projection，
  对持久节点的升级命令必须排在本轮普通编辑命令之前。
- 诊断面板按严重级别纵向使用完整宽度；诊断正文允许换行，不按固定三列压缩单条错误。
- Run 时间线每条事件必须同时提供距 `queuedAt` 的毫秒级相对偏移和 `occurredAt` 本机时间；相对时间
  用于定位节点停顿，完整绝对日期时间通过语义化 `time` 元素和悬浮提示保留。
- Run 时间线导出使用版本化 `yotta.run-timeline` JSON，必须导出完整 journal 而不是当前分页，并且
  只携带既有 Run 投影和脱敏 summary。模板自动化的 matched/timeout 状态与 adapter action 应累计
  `capture_ms`、`match_ms`、`capture_bytes` 和 `captures`，让性能问题可从导出记录分段定位。
- 模板匹配与画面变化观察使用进程内原始 RGBA 帧；PNG 编码只属于截图/图片产物边界，不能重新进入
  实时视觉热路径。点击模板默认使用首次匹配位置，只有用户显式设置非零 settle 时才重新抓屏定位。
- 原始帧通过 capture capability 传递时使用有界大分块，避免 1080p 帧产生上百次授权调用；Windows
  capture backend 已返回 `*image.RGBA`，adapter 不得再无条件复制完整帧。
- `versions:check` 只校验产品版本权威源、生成投影和明确的长期架构边界；一次性合同切换完成后删除
  清场扫描，禁止用裸子串搜索限制普通源码文本。各合同版本由所属 module、schema 与合同门禁验证。

## 内置节点分类审查

- 当前基线来自 tracked `contracts/node/current/builtin-authoring.json`，包含 146 个节点、21 个 Authoring
  分类；分类和数量是本次审查基线，不作为长期节点总数契约。
- 审查按分类逐项进行，每个节点同时检查标题、说明、图标、端口与配置字段、默认值与必填语义、编辑器
  可填写性、编译与 Program reopen、运行 adapter/capability、成功/失败路由以及用户可理解的错误反馈。
- 一个分类只有在全部节点完成检查、发现的问题已修复或明确记录、相关定向测试通过后才能在计划中完成。
- 开发阶段不反复运行全量门禁；所有分类完成后统一运行一次 `task check`。

## 子图管理交互

- 子图列表的高频操作是打开定义与在当前图创建调用：单击主行打开，拖到画布创建调用。
- 拖放不能成为唯一入口；行内保留可聚焦的“调用”按钮，并与拖放共用自身调用/循环依赖校验。
- 调用位置、重命名、复制与删除属于低频管理操作，统一收进行级更多菜单，不常驻占用列表宽度。
- 画布不再维护重复的全局“添加调用”菜单；添加节点与添加注释仍留在画布左上角。

## 键鼠宏编辑交互

- 录制和持久化继续使用按下、等待、松开等原子动作；默认编辑器将相邻且匹配的原子序列投影为带持续时间
  的按键或单击，原子视图作为高级入口保留。
- 语义行的修改、复制、删除和移动必须作用于它包含的完整原子动作组，不能打散按下/松开配对。
- 添加动作支持插入到指定行之后；多选支持整组复制和移动。拖放只从专用拖拽柄启动，输入框、下拉框和
  行选择不能意外触发拖拽。
- 鼠标按下和松开允许少量录制抖动后折叠为单击；明显位置变化视为拖拽并保留原子显示。

## AI 模型地址与协议

- HTTPS 供应商根地址是可保存的 API base URL；运行时只按用户显式选择的协议拼接固定路由。
- 支持 OpenAI Responses、OpenAI Chat Completions 和 Anthropic Messages，不根据域名猜测或回退协议。
- Chat Completions 当前只支持普通 Generate；Agent continuation 未实现前禁用 tool-calling 声明。
- 前端只镜像 Go 的必要 URL/传输安全边界，Go 仍是最终权威。
- API 密钥继续只写入操作系统凭据存储，endpoint 校验失败时不得尝试写入密钥。
- 模型档案的 `maxOutputTokens = 0` 表示不设置安装级上限；正数仍是安装上限，单次请求可声明更小的
  正数限制。没有任何限制时 provider payload 省略对应 token 字段。
- unverified 模型可执行普通 Generate/Extract；Agent、AI authoring 与工具权限必须继续验证 approved
  evaluation 和 exact current candidate，rejected/stale 模型不得进入 Host Profile。
- AI、HTTP、应用和自动化安装设置必须作为一个 generation 原子发布；设置保存后新 Run 立即使用新代，
  已在运行的 Run 保留其 lease，不要求重启也不允许 ambient fallback。
- admission readiness 必须保留 requirement ID；`model` requirement 的 target/credential 错误应说明
  具体 AI 槽位，不能使用“所需目标不可用”这类内部抽象。
- 画布 AI 分类只提供“生成文本”和“提取结构化数据”。受限 Agent 节点已删除；它的固定单工具 ToolSet
  与八项预算配置没有形成用户可兑现的独立能力，底层 Agent continuation 继续供 AI authoring 使用。
- 两个 AI 节点的提示词都使用多行文本编辑器。提取节点只让用户定义字段名、类型、说明与可空语义，
  后端从字段列表生成并验证严格 JSON Schema，不向普通用户暴露 schema 语言。
- 两个 AI 节点都提供可选 Image 输入，可直接连接窗口捕获或加载图片。运行时显式读取 Blob、限制源
  PNG 大小与像素数、缩放并编码为有界 JPEG，再按所选 provider 协议发送；不得忽略图像静默降级。
- OpenAI Chat Completions 的 API base URL 必须保留供应商要求的路径前缀，例如 NewAPI 常见的 `/v1`；
  Yotta 只在此前缀后追加 `/chat/completions`。HTTP 200 但正文是管理站 HTML 不是供应商拒绝，也不是
  User-Agent 封禁，应识别为“API 基础地址指向网页而非接口”并提示补全路径。
