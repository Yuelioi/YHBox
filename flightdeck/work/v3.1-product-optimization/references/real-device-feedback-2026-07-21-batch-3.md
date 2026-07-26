# 真机反馈第三批 — 2026-07-21

## Summary

| ID | 用户观察 | 期望与验收 |
| --- | --- | --- |
| FD-19 | 工作流内资源不能拖到画布 | 资源项提供明确 drag affordance，拖入画布创建/绑定对应节点；仍保留点击使用与键盘替代 |
| FD-20 | 精准录制只显示 `RECORDING_CALIBRATION_REQUIRED` | 错误面解释原因并提供直接进入目标校准的恢复动作，校准后可回到原录制上下文重试 |
| FD-21 | 属性的“清除”点击后消失，和“使用默认值”重复；required 参数语义不清 | 有默认值只提供恢复默认；无默认 optional 才允许清除；无默认 required 不提供不可完成的清除动作 |
| FD-22 | 点击模板已选模板图片，节点仍报缺少必填 binding | 资源选择必须写入 compiler 消费的同一 input binding；诊断定位到具体仍缺少的字段 |
| FD-23 | 写入日志的 message 不能在 Inspector 设置 | Inspector 提供 message 文本配置；可选 observable data edge 连接后覆盖配置值，保留记录任意值的能力 |
| FD-24 | 首页与工作流内资源库不是标准多项展示 | 首页资源库和 Workflow Resource Dock 共享紧凑、可扫描、支持搜索/类型筛选的多项列表语言 |
| FD-25 | Inspector 隐藏按钮位置差，且选择节点会强制重新显示 | 提供持久 Inspector 总开关；用户关闭后选择/定位节点不自动打开，显式重新开启才恢复 |
| FD-26 | 改名是孤立 icon；缺少设置与更多菜单 | 保存旁提供设置入口承载名称/描述/分类/tags；更多菜单至少提供重新加载并保留后续扩展 seam |
| FD-27 | 新建子图后空态遮挡入口，入口端口文字重叠，接口刷新不可用且错误晦涩 | 新子图初始拓扑可编辑且不被遮挡；boundary 端口布局清晰；接口刷新在前置条件不足时禁用并说明，满足后可用 |
| FD-28 | 精准录制保存页过度复杂、时间线名称误导、不能鼠标裁剪、保存后重复打开；InputClip 回放只显示泛化失败 | 保存页聚焦裁剪与名称；原始事件默认折叠为诊断详情；时间线明确不执行回放并支持鼠标选边界；每个 pending 只打开一次；跟随活动档案的目标可回放，失败自动打开运行时间线 |
| FD-29 | 简易/精准录制点击后立即进入倒计时，来不及切回目标；各录制快捷键语义不明确 | 两种模式先进入待开始，按统一开始键后倒计时 3 秒再采集；默认 F10 开始、F11 暂停/继续、F12 停止，均由快捷键中心配置 |
| FD-30 | 精准录制绕花坛一圈，回放路径半径由约 20 放大到约 25，不能回到原位；3.0 基本可 1:1 复刻 | 回放必须服从 Clip 的绝对单调时间轴，逐事件 RPC/目标解析/注入耗时不得累计进下一事件；高密度 RawDelta Clip 的实际时长应贴近 nominal duration |
| FD-31 | InputClip 回放看不到录制时的 counts/360，也无法判断当前目标采用什么校准 | Inspector/资源详情只读展示录制源与本机目标 counts/360；runtime 自动换算，不提供会破坏精准复刻的手工倍率 |

## UI decisions

- 资源列表复用现有暗色密集工具视觉；使用 Tabler outline 图标、稳定行高和清晰 hover/focus，不引入新的卡片风格。
- Inspector 的 open/closed 是用户偏好，selection 只更新属性上下文，不覆盖偏好。
- 子图空态不覆盖可交互节点；固定工具条必须为画布内容预留空间。

## Implementation result

- FD-19 / FD-24：主页和编辑器资源坞复用 `AssetLibraryList`；资源拖放使用有版本边界的自定义 MIME payload，落点决定新节点位置，双击/加号仍可用。
- FD-20：同时识别结构化错误码和 Wails 只返回 `Error.message` 的形态，错误 Toast 提供直达“输入与校准”的动作。
- FD-21：binding action policy 只显示一个可完成动作：有默认值时恢复默认；无默认 optional 时清除；required 不提供清除。
- FD-22 / FD-23：现场 Source 中模板 blob binding 已存在，实际缺失项是日志 message；日志输入改为 optional observable，并增加 Inspector message 配置，连线值优先。
- FD-25：Inspector 开关移到主工具栏并持久化；选择、右键选择和定位节点都不再覆盖关闭偏好。
- FD-26：保存旁增加工作流设置（名称、描述、分类、tags）与更多菜单；重新加载会保护未保存改动。
- FD-27：主图才显示可交互大空态；空子图使用不拦截指针的底部提示，boundary 标签为 handle 留出空间；接口推断在前置条件不足时禁用。

## Verification

- Frontend targeted: 4 files / 30 tests passed.
- Frontend `vue-tsc --noEmit`, changed-file Oxlint/ESLint and i18n parity/compile/reference checks passed.
- Go: `internal/nodes` and `internal/noderuntime` focused Log tests passed.
- Node/Workflow contracts regenerated through `task contracts:update`.
- Pending: Windows WebView / real-device smoke for FD-19 through FD-27.

## FD-20 real-device follow-up

首次复测仍出现 `RECORDING_CALIBRATION_REQUIRED`。现场设置显示：全局活动鼠标档案“异环”为
4134 counts/360°，但自动化目标 `window-target` 的 `mouseCounts360` 为 0。

根因不是校准未保存，而是录制服务只读取已安装自动化目标的回放校准值，没有在目标未覆盖时读取
`Settings.ActiveMouseCounts360()`。原 Toast 只返回错误码作为 `Error.message`，恢复按钮虽然出现，描述仍是裸码。

修复后的优先级：

1. 自动化目标显式自定义 counts360；
2. 当前活动鼠标校准档案；
3. 0（未校准，精准相对录制继续拒绝）。

录制 composition 在每次 Acquire 时读取活动档案，因此切换或重新校准后无需改写目标身份；目标显式值仍
优先。自动化目标设置页把原始数字框改为“跟随当前档案 / 此目标自定义”模式，并显示正在继承的档案和
计数。裸错误码 message 现在也通过现有 `error.*` 字典本地化。

定向验证：前端 3 个文件/29 项测试、类型与 i18n 检查通过；Go `internal/desktopapp` 和
`internal/services/recording` 相关测试通过。等待重新构建后的 Windows 真机精准录制复测。

## FD-28 precise recording journey follow-up

第二次真机复测确认录制产物已经带有 4134 counts/360°，但回放 provider 仍只读取目标 Profile 中的
`mouseCounts360=0`，因此“跟随当前档案”只修复了录制、没有贯通回放。与此同时，停录返回值和
`recording.state.pending` watcher 都会打开保存页；运行失败后工作台又自动切到只含泛化系统消息的日志页。

修复后：

- 活动校准作为运行期覆盖注入 automation installation，不进入 Profile digest 或 workflow consent；目标显式
  counts360 仍优先，活动档案变化会触发 installation generation 更新。
- 停录只由权威 pending 状态打开保存页，移除 RPC 返回值的第二条打开路径。
- “预览时间线”改为“事件时间线”，明确不会操作目标；鼠标单击会移动最近的裁剪边界，毫秒输入仍保留。
- 原始事件与分辨率等诊断信息默认折叠；保存页移除精准模式的重复摘要并取消强制 92vh 高度。
- Run 失败自动打开时间线而不是日志，直接显示本地化 failure code、节点与 attempt。

定向验证：Go `internal/automation/installed`、`internal/services`、`internal/desktopapp` 通过；前端 5 个文件/
44 项测试、类型、i18n、变更文件格式与 lint 通过。等待 Windows 真机复测录制 → 保存 → 插入 → 回放全链路。

### FD-28 third real-device pass

复测发现鼠标选择的裁剪起点落在长按键区间时，保存仍返回英文
`trim starts inside a held key interval`。这证明“前端任意选点、后端要求安全切点”的组合本身不可完成；
保存页把描述、分类和标签折叠也错误降低了资源入库主流程的优先级。工作流编辑器中的分类/标签还使用
普通文本框，没有复用资源库已经存在的可创建选择器。

修正后的语义：

- 裁剪可以落在任意时间。起点若已有按键或鼠标按钮处于按下状态，生成的 Clip 在 0 时刻补按下；终点
  仍处于按下状态时补释放，并重新编号事件，保证回放不会缺少起始状态或留下卡键。
- 描述、分类、标签直接显示；只有原始事件等诊断数据折叠。
- 编辑器保存录制与资源库统一使用可创建 `UInputMenu`：加载同资源类型的 category/tag facets，输入新值后
  通过创建项确认，标签以多选值保存，不再解析逗号文本。

定向验证：`internal/services/recording` 全包通过；前端 3 个相关测试文件/30 项测试、类型、i18n 与
变更文件 Oxlint 通过。

## FD-29 recording preparation and hotkeys

简易录制和精准录制共用后端权威状态流：`armed → countdown → recording`。点击录制只锁定目标并打开
悬浮控制窗，不采集输入；用户切回目标窗口后按配置的开始键，或点击悬浮窗按钮，才进入 3 秒倒计时。
开始键在倒计时前由独立低级键盘 hook 拦截，因此不会落入录制数据。

默认快捷键统一为 F10 开始、F11 暂停/继续、F12 停止，避免占用现有 F8 鼠标校准和 F9 窗口截图；
三个动作均注册到快捷键中心，可由用户修改。准备或倒计时阶段可从悬浮窗、编辑器工具栏或资源库取消，
取消会失效倒计时 generation 并释放目标 lease，过期 timer 不能再次启动 recorder。

## FD-30 precise playback timing regression

现场配置和产物排除了“校准缩放错误”作为主因：活动鼠标档案为 4132 counts/360°，目标使用 `sendinput`
并跟随活动档案，相关 Clip 的 source counts 也是 4132，因此 provider 缩放系数为 1.0。24.375 秒的疑似
绕行样本含 3526 个事件（3492 个 RawDelta），水平累计 -4072 counts，已经接近一整圈，采集/编码没有
把转向量缩成约 80%。Windows Raw Input 采集实现与 3.0 基本相同，3.1 还改用原生事件 tick，采集侧不是
本次明显回归点。

真实 Run 记录直接证明 3.1 时间轴随事件密度膨胀：

| Events | Clip nominal | Node actual | Stretch |
| ---: | ---: | ---: | ---: |
| 653 | 15.266 s | 16.847 s | 10.4% |
| 1504 | 10.984 s | 13.627 s | 24.1% |
| 1682 | 7.578 s | 10.194 s | 34.5% |
| 3420 | 6.891 s | 11.400 s | 65.4% |
| 4185 | 16.000 s | 22.425 s | 40.2% |

根因是 3.1 `playInputClip` 按相邻事件的原始 delta 逐次调用 `Wait(delta)`，却没有扣除上一事件的同步
开销；每条事件还跨 resource/provider 边界做 JSON artifact、重新枚举并解析 exact executable window、
`BringToFront` 和 SendInput。约 1.3–2.4 ms 的逐事件成本因此永久累加。键盘保持和鼠标转向被一起拉长，
总 RawDelta 虽保持不变，单位时间转向速度下降，边走边转的曲率降低、半径放大。

3.0 使用 QPC 单调时钟和 `start + event.TUs` 绝对 deadline；前一事件变慢时下一等待会自动缩短或追赶，
并且运行期持有已解析 HWND，不会为每个 RawDelta 重新扫描目标。修复应恢复等价的绝对时间语义，并把
目标解析/前台激活收敛到 playback session；长期可让 provider 接受带时间戳的事件批次，在注入边界完成
高精度调度。回归测试需用约 1500 个 16 ms RawDelta、注入固定逐事件成本，断言完成时间仍接近 Clip
duration，而不是 `duration + eventCount × cost`。

## FD-31 playback calibration visibility and authoring control

当前 counts/360 并未丢失。录制服务从所选自动化目标解析有效校准值，写入不可变 InputClip carrier 的
`ClipMeta.mouseCounts360`；回放将它作为 source counts 传给 provider，再用目标自定义值或当前活动校准
作为 target counts，按 `target / source` 缩放 RawDelta。资源 Summary 已解码并返回这份 metadata，但普通
资源卡和“回放输入录制”节点 Inspector 没有展示，因此用户无法核对回放采用了什么值。

源 counts 是录制事实，不应成为可随节点修改的普通输入；目标 counts 是设备/游戏目标的本机配置，也
不应固化进可移植工作流。资源详情和节点 Inspector 只读展示“录制源 counts/360”“当前目标
counts/360”；runtime 对 RawDelta 严格使用 `target / source` 自动换算。曾实现的 `turn-scale` 会让精准
复刻变成手工近似并制造不必要的 Node Contract 摘要，因此已撤回。

### FD-30 / FD-31 implementation

- Executor 为 adapter 提供独立于审计墙钟的单调时钟；InputClip 以 session start 为原点，对每条事件等待
  `start + event.TUs - now`，上一事件的 provider/注入耗时会缩短下一次等待，不再累计拉长 Clip。
- playback provider 在 Open 时完成 profile 验证并打开平台 playback driver；Windows 只解析并置前目标
  一次，session 内复用固定 HWND；Android 同样在 Open 时固定已解析设备目标。
- 节点 Inspector 异步读取绑定 Clip 的源 counts，结合节点有效 target 和活动校准展示源 counts、目标
  counts 与来源；资源工作台顶部直接显示录制源 counts，不再藏在原始事件详情中。
- 1500 个、间隔 16 ms 的 RawDelta 回归测试为每次 Play 注入固定 2 ms 成本，完成时间严格等于 nominal
  duration 加最后一次注入成本；旧相对等待实现会额外累计约 3 秒。
- 节点契约恢复稳定摘要 `5c353fb…`。开发期 `ff7ea9…` 节点在没有连接临时端口时移除
  `turn-scale` binding 并迁回稳定契约；未登记摘要仍拒绝自动替换。
- 使用真实 Source 形状建立应用级回归：迁移前同时产生 `NODE_CONTRACT_MISMATCH` 和派生
  `UNKNOWN_PORT`，迁移并重新编译后两项都必须消失。

定向验证：Go `internal/noderuntime`、`internal/nodes`、`internal/automation/installed`、
`internal/workflow/compiler`、`internal/workflow/authoring`、`internal/nodeauthoring` 和
`internal/appbootstrap` 通过；前端 6 个相关测试文件/68 项测试、typecheck、i18n、变更文件 lint 通过；
production bundle gate 通过，editor gzip 217486/220000 bytes。待 Windows 真机复测 nominal/actual
duration、绕行半径和终点，并核对录制源/本机目标 counts/360。

兼容性补充验证：authoring/application/workflow service 定向 Go 测试通过；EditorSession 旧摘要自动升级
测试、typecheck、i18n、变更文件 lint 与 production bundle 通过，editor gzip 218155/220000 bytes。
未触发 Rust 或全仓 coverage。

### Online workflow portability boundary

当前 `.yotta-workflow` 已将完整 Workflow Source 和全部内容寻址 Blob 打包，因此 InputClip 及录制源
counts/360 可以无损分发。目标 counts、credential 和设备身份属于接收方机器，不应写进节点参数或 Blob。
现有 Bundle 仍缺少精确 Node Package 依赖预检和导入后的本机 target/credential 重绑定；完成这些能力前，
下载工作流只能视为“资源完整但需要本机适配”，不能静默替换契约或直接运行。
