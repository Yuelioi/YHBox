# 鼠标轨迹自动化：同类产品证据与 Yotta 设计结论

**Date**: 2026-08-04

本文只采用产品官方文档、官方源码和 Windows 平台规范。结论用于区分普通键鼠宏、拖拽/轨迹和前后台输入，不代表对任意游戏兼容性的承诺。

## 结论先行

1. 成熟产品通常不把每个 `mouse-move` 展开成一个画布节点。默认模型是 `Move`、`Click`、`Drag` 等语义动作；完整轨迹是显式开启的高级录制能力。
2. `mouse-down + 时间 + mouse-up` 只有在按住期间没有形成拖拽时才可以折叠为 `Click(duration)`。Windows 自身也使用可配置的 `SM_CXDRAG/SM_CYDRAG` 矩形区分点击抖动和拖拽，而不是要求按下、松开坐标逐像素相等。[Microsoft DragDetect](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-dragdetect)、[GetSystemMetrics](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-getsystemmetrics)
3. “保存了完整轨迹”和“能够后台发送”是两个正交能力。完整轨迹描述做什么；SendInput、窗口消息、可访问性 API、浏览器 API 或驱动决定怎样送到目标。
4. 对 Yotta，合理产品形态是：普通录制生成可编辑 Macro；精准录制生成一个折叠的 InputClip 资源，并由单个“精准输入回放”节点执行。不要新增成百上千个轨迹点节点。

## 同类产品怎么做

| 产品 | 普通动作模型 | 轨迹、拖拽和时间 | 后台边界 |
| --- | --- | --- | --- |
| Razer Synapse | 宏录制按顺序保存按键、按钮和可选鼠标移动；用户可在录制前选择是否记录鼠标移动。 | 官方提供 `None`、屏幕绝对坐标、前景相对坐标和以当前光标为起点等策略；旧版官方手册还允许修改总时长并按比例缩放宏内全部延迟。这证明完整轨迹适合作为显式录制选项，而不是普通宏的强制内容。[Synapse 4 官方帮助](https://mysupport.razer.com/app/answers/detail/a_id/1483)、[官方产品手册](https://dl.razerzone.com/master-guides/RazerSynapse4/PROCLICKV2VERTICALEDITION-00000199-en.pdf)、[总时长行为](https://dl.razerzone.com/master-guides/BW2019/BW2019OMG-ENG.pdf) | 这是面向真实桌面输入的设备宏；官方材料没有把鼠标轨迹描述为任意遮挡/最小化窗口的消息级后台输入。 |
| Power Automate Desktop | Recorder 按 UI 元素把鼠标和键盘行为转换成一个个动作，而不是暴露原始高频采样流。 | Recorder 能识别拖放相关步骤并生成移动/调整窗口等语义动作；当前仍不支持录制通用的“拖放窗口 UI 元素”动作。手工 `Move mouse`/`Send mouse click` 使用目标坐标和“瞬移/低中高速动画”，UI Automation 另有独立 `Drag and drop UI element`。[录制器](https://learn.microsoft.com/en-us/power-automate/desktop-flows/recording-flow)、[鼠标键盘动作](https://learn.microsoft.com/en-us/power-automate/desktop-flows/actions-reference/mouseandkeyboard)、[UI Automation 动作](https://learn.microsoft.com/en-us/power-automate/desktop-flows/actions-reference/uiautomation) | 物理鼠标动作在非交互会话会报错，图像搜索也要求画面可见；这类动作不等于后台。元素级 UI Automation 能绕开一部分物理输入需求，但仍取决于应用暴露的自动化技术。[鼠标键盘动作异常定义](https://learn.microsoft.com/en-us/power-automate/desktop-flows/actions-reference/mouseandkeyboard) |
| UiPath | `Click`、`Hover`、`Drag and Drop` 是独立语义活动；同时保留 single/double/down/up 等低级点击类型。 | Click 提供 `Instant`/`Smooth`，Drag and Drop 保存源、目标、按钮、移动类型及动作间延迟，不要求保存人工原始轨迹。[Click](https://docs.uipath.com/activities/other/latest/ui-automation/click)、[Drag and Drop](https://docs.uipath.com/activities/other/latest/ui-automation/n-drag-and-drop) | 官方明确拆分 Hardware Events、Window Messages、Simulate、Chromium API：Hardware Events 兼容性最高但不能后台；Window Messages 和 Simulate 可后台但有兼容边界，复杂图像活动仍可能回到前台。[输入方法矩阵](https://docs.uipath.com/activities/other/latest/ui-automation/input-methods) |
| AutoHotkey v2 | `MouseMove` 保存“终点坐标 + 速度”，`MouseClickDrag` 保存起点、终点、按钮、速度；也保留显式 Down/Up。 | `MouseMove`/`MouseClickDrag` 的速度只在 SendEvent 下产生可见的渐进移动，SendInput 会忽略速度并瞬移。任意曲线需要脚本显式发多个移动点，不是官方逐点录制格式。[MouseMove 官方源码文档](https://github.com/AutoHotkey/AutoHotkeyDocs/blob/v2/docs/lib/MouseMove.htm)、[MouseClickDrag 官方源码文档](https://github.com/AutoHotkey/AutoHotkeyDocs/blob/v2/docs/lib/MouseClickDrag.htm) | 普通 Send 面向活动窗口；后台另走 `ControlClick`，可按 HWND/控件/客户区坐标发送且可避免激活，但官方明确说明不适用于所有窗口和控件。[ControlClick](https://github.com/AutoHotkey/AutoHotkeyDocs/blob/v2/docs/lib/ControlClick.htm)、[Send](https://github.com/AutoHotkey/AutoHotkeyDocs/blob/v2/docs/lib/Send.htm) |
| SikuliX | 以可见图像/区域为目标，主动作是 click、hover、dragDrop。 | 默认根据端点和 `MoveMouseDelay` 合成连续移动；复杂拖拽可用 `drag()`，在按住期间插入一个或多个 `mouseMove()`/`hover()`，最后 `dropAt()`。还保留 `mouseDown/mouseUp` 低级动作。[Region 鼠标动作](https://sikulix-2014.readthedocs.io/en/latest/region.html#acting-on-a-region)、[脚本时序设置](https://sikulix-2014.readthedocs.io/en/latest/scripting.html) | SikuliX 的定位和操作面向真实或等价虚拟屏幕上的可见 GUI，不是 PostMessage 式任意遮挡/最小化窗口后台输入。[官方场景说明](https://sikulix-2014.readthedocs.io/en/latest/scenarios.html) |
| 按键精灵 | 官方资源站的示例把前台 `MoveTo` 与 `Plugin.Bkgnd.MoveTo/LeftClick/KeyPress` 分成不同命令族。 | 官方材料能证明它保留移动、点击和 Delay 等脚本原语，但没有找到可信的一手资料证明普通录制会保存高频、逐点原始轨迹。[官方后台示例](https://zy.anjian.com/lab/content_06.html)、[官方命令目录](https://zy.anjian.com/sitemap.html) | 官方 FAQ 说明后台能力由独立窗口插件提供；官方教程也建议内置后台插件失效时更换目标专用插件。因此“按键精灵支持后台”不等于一个 PostMessage 实现兼容所有应用。[官方 FAQ](https://www.anjian.com/download.htm)、[官方论坛后台教程](https://bbs.anjian.com/showtopic-700169-1.aspx) |

## OK Script / OK脚本专项核实

核实对象是 `ok-oldking/ok-script` 官方仓库；其 README 同时链接仓库内 API 文档和同名 PyPI 包，PyPI 的项目主页也指回该仓库。本节固定在仓库提交 `c7fbefb45df3e71e760451ce62671c0a461401b2`（2026-08-03）及当时 PyPI 最新版 `1.0.181`，避免后续源码变化使结论失去上下文。[官方仓库 README](https://github.com/ok-oldking/ok-script/blob/c7fbefb45df3e71e760451ce62671c0a461401b2/README.md#L52-L66)、[官方 PyPI 1.0.181](https://pypi.org/project/ok-script/1.0.181/)

- **录制器不录鼠标轨迹，也不生成拖拽。** 它创建 `pynput.mouse.Listener` 时只注册 `on_click`，没有注册 `on_move`；按钮松开后，先前的 `mouse_down` 会被折叠成带 `down_time` 的 `click`。代码生成阶段也只接受 click 和键盘事件并输出 `click_relative`，所以当前官方录制器无法从人工操作恢复自由移动路径或按住期间的拖拽路径。[监听器初始化](https://github.com/ok-oldking/ok-script/blob/c7fbefb45df3e71e760451ce62671c0a461401b2/ok/gui/tasks/RecordScript.py#L42-L62)、[点击折叠](https://github.com/ok-oldking/ok-script/blob/c7fbefb45df3e71e760451ce62671c0a461401b2/ok/gui/tasks/RecordScript.py#L136-L161)、[代码生成过滤](https://github.com/ok-oldking/ok-script/blob/c7fbefb45df3e71e760451ce62671c0a461401b2/ok/gui/tasks/RecordScript.py#L284-L299)
- **公开动作只有一次移动和端点滑动，没有“轨迹模式”参数。** `move(x, y)` 只接收终点；`swipe(from_x, from_y, to_x, to_y, duration=0.5, after_sleep=0.1, settle_time=0)` 接收起终点和时间。`duration` 是期望的手势秒数，调用后端前乘以 1000 转成毫秒；`after_sleep` 是后端返回后再等待的秒数；`settle_time` 被传给后端，意图是在终点松开前停留，但并非所有后端都使用它。源码和官方 API 面上均没有 `uniform`、`bezier`、`curve`、`instant` 一类枚举或模式选择参数。[`move` API](https://github.com/ok-oldking/ok-script/blob/c7fbefb45df3e71e760451ce62671c0a461401b2/ok/task/task.py#L395-L400)、[`swipe` API](https://github.com/ok-oldking/ok-script/blob/c7fbefb45df3e71e760451ce62671c0a461401b2/ok/task/task.py#L257-L278)
- **所谓“瞬移”是 `move` 的单次目标移动行为，不是一个命名模式。** PostMessage 后端只投递一次 `WM_MOUSEMOVE`，pynput 后端只设置一次 `controller.position`，浏览器后端也只调用一次 `page.mouse.move(x, y)`；没有公开的时长或缓动参数。[PostMessage `move`](https://github.com/ok-oldking/ok-script/blob/c7fbefb45df3e71e760451ce62671c0a461401b2/ok/device/interaction_methods/post_message.py#L73-L76)、[pynput `move`](https://github.com/ok-oldking/ok-script/blob/c7fbefb45df3e71e760451ce62671c0a461401b2/ok/device/interaction_methods/pynput.py#L106-L113)、[browser `move`](https://github.com/ok-oldking/ok-script/blob/c7fbefb45df3e71e760451ce62671c0a461401b2/ok/device/interaction_methods/browser.py#L130-L134)
- **“匀速”只能描述部分后端的线性插值实现，不能写成产品模式。** PostMessage、pynput、pydirect 和原神专用后端都按相同坐标步长、每步约 10 ms 移动；浏览器后端也做直线等分，并按请求时长分配每步等待。前四者当前用 `steps = int(duration / 100)` 配合 10 ms 等待，因此仅移动阶段约为传入毫秒数的十分之一，并不能准确兑现公开层的 `duration`。pynput 单独把非正步数钳为 1；另外三个实现没有该保护，传入不足 100 ms 时会在计算步长处除以零。[PostMessage 线性滑动](https://github.com/ok-oldking/ok-script/blob/c7fbefb45df3e71e760451ce62671c0a461401b2/ok/device/interaction_methods/post_message.py#L99-L123)、[pynput 线性滑动](https://github.com/ok-oldking/ok-script/blob/c7fbefb45df3e71e760451ce62671c0a461401b2/ok/device/interaction_methods/pynput.py#L115-L141)、[pydirect 线性滑动](https://github.com/ok-oldking/ok-script/blob/c7fbefb45df3e71e760451ce62671c0a461401b2/ok/device/interaction_methods/pydirect.py#L78-L98)、[原神后端线性滑动](https://github.com/ok-oldking/ok-script/blob/c7fbefb45df3e71e760451ce62671c0a461401b2/ok/device/interaction_methods/genshin.py#L166-L193)、[browser 线性滑动](https://github.com/ok-oldking/ok-script/blob/c7fbefb45df3e71e760451ce62671c0a461401b2/ok/device/interaction_methods/browser.py#L171-L187)
- **贝塞尔曲线确实存在，但只接入 Nemu 模拟器触控路径。** 内部 `insert_swipe(p0, p3, speed=15, min_distance=10)` 生成随机三次贝塞尔曲线：`speed` 表示平均每 10 ms 的像素数，并通过 `segments = max(int(distance / speed) + 1, 5)` 控制采样分段，数值越大，固定距离下的点越少；两个控制点在直线三等分附近随机扰动，采样在起止位置较密、中央较疏，`min_distance` 用于过滤相邻过近的点。[贝塞尔生成器](https://github.com/ok-oldking/ok-script/blob/c7fbefb45df3e71e760451ce62671c0a461401b2/ok/device/interaction_methods/swipe.py#L5-L63) 交互层只有 `ADBInteraction.swipe_nemu` 调用该生成器，并以每点 10 ms 下发；传入该函数的公开 `duration` 没有参与曲线生成或时间控制。另一个 Nemu IPC 实现也只在 `swipe_nemu_ipc`/`drag_nemu_ipc` 中使用它，drag 将 `speed` 改为 20。[ADB Nemu 调用](https://github.com/ok-oldking/ok-script/blob/c7fbefb45df3e71e760451ce62671c0a461401b2/ok/device/interaction_methods/adb.py#L43-L58)、[Nemu IPC 调用](https://github.com/ok-oldking/ok-script/blob/c7fbefb45df3e71e760451ce62671c0a461401b2/ok/capture/adb/nemu_ipc.py#L549-L568)
- **同一个 `swipe(duration)` 的时间语义随后端变化。** Nemu 贝塞尔分支忽略 `duration`；uiautomator2 分支明确注明该参数影响有限，并按距离每 16 像素一个直线点快速移动；ADB shell 分支才把毫秒数原样交给 Android `input swipe`。这说明 OK Script 当前选择的是“一个语义动作、后端自行近似”，而不是可跨后端复现的录制轨迹或统一速度曲线。[ADB 分支实现](https://github.com/ok-oldking/ok-script/blob/c7fbefb45df3e71e760451ce62671c0a461401b2/ok/device/interaction_methods/adb.py#L60-L108)

检索范围包括官方 README、PyPI 1.0.181、生成的 API 文档、录制器源码、`BaseTask` 输入接口、`ok/device/interaction_methods/` 下的全部交互后端、Nemu IPC 触控实现，以及同作者的 [`ok-script-app` 模板仓库](https://github.com/ok-oldking/ok-script-app)。在这些一手资料中没有找到面向用户的“匀速 / 贝塞尔 / 曲线 / 瞬移”模式开关或 `mode`/`easing`/`path` 类型参数，也没有找到鼠标轨迹录制格式。可以确认的是上述具体源码行为；不能据此推断某个未公开发行版或下游脚本应用没有自行增加 UI 选项。

## Windows 为什么无法用一条后台方案覆盖所有目标

- `PostMessage` 只把指定消息放进目标窗口所属线程的消息队列；它不是系统键鼠输入流。消息投递还受 UIPI 完整性级别约束。[PostMessage](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-postmessagea)
- `SendInput` 把鼠标移动、按钮和按键事件串行插入系统输入流，也受 UIPI 约束。它更接近真实前台输入，但会占用全局桌面输入和真实光标。[SendInput](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-sendinput)
- Raw Input 是另一套模型，目标注册设备后通过 `WM_INPUT` 读取原始 HID 数据；传统的 `WM_MOUSEMOVE/WM_KEYDOWN` 消息与 Raw Input 不是同一条数据路径。[Raw Input Overview](https://learn.microsoft.com/en-us/windows/win32/inputdev/about-raw-input)
- Windows 默认可以合并 `WM_MOUSEMOVE`；`MOUSEEVENTF_MOVE_NOCOALESCE` 才要求 SendInput 产生的移动消息不合并。因此“录了 1000 个点”并不意味着目标会逐点观察到 1000 个窗口消息。[MOUSEINPUT](https://learn.microsoft.com/en-us/windows/win32/api/winuser/ns-winuser-mouseinput)、[Mouse Input Overview](https://learn.microsoft.com/en-us/windows/win32/inputdev/about-mouse-input)
- 读取 `GetAsyncKeyState`、Raw Input 或自有轮询状态的目标，可能完全不理会人工构造的普通窗口消息。是否支持后台最终必须按目标输入技术验证。[GetAsyncKeyState](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-getasynckeystate)

## 录制精度和数据量

- 普通 UI 自动化的正确“精度”是目标、按钮、动作类型和相对时序正确，不是无条件保存鼠标硬件的每一个采样点。Power Automate、UiPath、AutoHotkey 和 SikuliX 的主动作都证明端点/元素和时长足以覆盖大多数点击与拖拽。
- 只有相机转向、绘图、签名、手势识别或路径避障等场景需要完整轨迹。绝对路径关注几何误差，RawDelta 路径关注每个相对 count 的累计值，两者不能使用相同的压缩规则。
- 微软把 Raw Input 的 buffered 读取明确定位给 1000 Hz 鼠标等高频设备；精准相对输入若依赖 Raw Input，应批量排空缓冲区，不能只保留消息循环来得及处理的最后一点。[Raw Input Overview](https://learn.microsoft.com/en-us/windows/win32/inputdev/about-raw-input)
- 保存格式应包含单调时间戳和同时间戳下的稳定顺序。整体变速只缩放事件间隔，不改变 Down/Move/Up 次序、RawDelta 总和或最终落点。

## 给 Yotta 的最终模型

### 普通录制（默认）

- 自由鼠标移动只用于识别点击和拖拽，不持久化为高频动作；宏顶层 `meta.autoMove` 决定运行时是否在 `Click` 前补一个语义 `Move`。
- 保存 `Click(point, button, duration)`、`Scroll`、按键和等待。
- 从 mouse-down 开始暂存按住期间的移动点：没有越过拖拽阈值则折叠为 Click；越过阈值则生成 `Drag(start, end, duration, path?)`。
- Windows 上使用系统 `SM_CXDRAG/SM_CYDRAG` 作为点击/拖拽判定基线，允许正常手抖；不要要求 down/up 坐标完全相等。
- 普通 UI 拖拽默认只保留端点、持续时间和移动曲线；只有轨迹确实影响结果时才保留少量关键路径点。

### 精准录制（显式开启）

- 保存绝对 `MouseMove`、相对 `RawDelta`、按键/按钮、滚轮及原始时间线。
- 作为一个不可变 InputClip 资源回放，画布上只出现一个回放节点；编辑器提供时间轴、轨迹预览、裁剪和整体速度缩放，不把采样点展开成节点。
- 绝对轨迹可在保证误差上限的前提下做折线简化；相机转向用的 RawDelta 不应做会改变累计位移的几何简化。
- 回放速度应缩放相邻事件时间差；按钮 Down/Up 的相对顺序和最终累计位移必须保持。

### 后端能力

- `sendinput`：支持真实绝对轨迹、拖拽和相对位移；是精准轨迹的主要 Windows 后端。
- `postmessage`：可向兼容窗口发送客户区移动、Down/Up/Click，但只能承诺窗口消息语义。目标读 Raw Input、异步键鼠状态或真实 cursor 时必须明确报“不支持/需前台后端”，不能静默宣称精准回放。
- 元素/模板点击：优先保留 Click/Drag 语义，让每个 adapter 选择可访问性 API、浏览器协议、Android gesture、窗口消息或 SendInput；不要过早展开成原子消息。

## Yotta 实施结果（2026-08-04）

仓库现在采用一套共享运动模型和两层录制数据：

- [`pointermotion/plan.go`](../../../internal/automation/pointermotion/plan.go) 统一规划即时、匀速直线和确定性三次贝塞尔采样，Windows 前台与后台 adapter 只负责投递采样点。
- [`macro/from_input.go`](../../../internal/services/macro/from_input.go) 不把普通录制的自由移动写进动作列表；它将按下—移动—松开折叠为 `Drag`，并把未越过 4 像素阈值的抖动折叠为 `Click`。
- [`macro/plan.go`](../../../internal/services/macro/plan.go) 在执行规划阶段按持久化的 `meta.autoMove` 为 `Click` 补 `Move`；已知上一语义位置距目标不足 5 个录制分辨率像素时跳过，`Drag` 不受该策略影响。
- [`inputclip/model.go`](../../../internal/services/inputclip/model.go) 继续承载精准模式的绝对 `MouseMove`、相对 `RawDelta` 和完整微秒时间线，不把采样点展开成工作流节点。
- “移动指针”和“拖拽指针”工作流节点、宏中的显式 `Move`/`Drag`，以及点击前自动移动策略，共享 `instant`、`linear`、`bezier` 三种运动方式；普通录制默认启用 300 ms 的 `linear` 自动移动，用户可在宏编辑器中修改或关闭。

这保持了普通宏的可读、可编辑语义，也保留了精准轨迹能力；后台输入是否生效仍取决于目标是否消费传统窗口消息，而不是轨迹数据是否原子化。
