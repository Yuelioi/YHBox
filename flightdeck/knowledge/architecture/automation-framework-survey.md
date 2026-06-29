# 自动化框架调研 — YHFish 大升级前的对标基线

SUMMARY: ok-script / MaaFramework / Airtest / RPA / Appium 等自动化框架调研，以及对 YHFish 的取舍结论
READ WHEN: 设计 Target/Controller 架构 / 支持 Android 或浏览器 / 处理后台点击、截图坐标、输入后端兼容性 / 做大升级决策前
RECHECK WHEN: 引入新控制后端 / 改窗口或设备抽象 / 改节点执行 trace / 评估是否接入 MaaFramework、Airtest、ADB、Rust native controller 时

---

## 先说结论

YHFish 不应该把现有系统整体替换成 MAA、Airtest 或商业 RPA 的形状。更合理的方向是：

- **保留 YHFish 的可视化节点、容器、资产、调度和本地产品体验**。
- **把底层升级成多目标自动化内核**：`Target -> Controller -> Screenshot/Input -> Perception -> Action Trace`。
- **Go 继续做主运行时**，Rust 只下沉 Win32 / native hot path。
- **Android 应该作为一等 Target 接入**，但先走 YHFish 自己的 `AdbController`，不要一开始绑定 MaaFramework。

这次 After Effects 的 `Ctrl+N`、新建合成弹窗、截图取点仍落在主窗口的问题不是孤例。它暴露的是：当前系统还把“窗口”“截图来源”“输入后端”“点击坐标系”混在一起，没有一等 `Target/Controller/CoordinateSpace`。

## 调研对象

### ok-script

源码：<https://github.com/ok-oldking/ok-script>

定位：Python 图像识别自动化框架，面向 Windows 游戏、ADB、模拟器、浏览器等目标。它包含 UI、截图、输入、设备控制、OCR、模板匹配、调试浮层、测试、打包、升级、多语言。

值得学：

- `DeviceManager` 同时管理 Windows、ADB、浏览器目标。
- capture 和 interaction 分离，Windows 下有多种截图/输入后端。
- 任务基类提供 `click`、`send_key`、`ocr`、`find_one`、`box_of_screen` 等统一 API。
- 分辨率比例适配、相对坐标、OCR 修正、调试截图都内置在 task 操作层。

不适合直接照搬：

- Python 生态和 YHFish 的 Go/Wails 主体差异大。
- AGPL-3.0 许可证不适合直接混入未来可能闭源或商业分发的核心。
- 它更偏脚本/任务框架，不是 YHFish 这种可视化节点产品。

### ok-wuthering-waves

源码：<https://github.com/ok-oldking/ok-wuthering-waves>

定位：基于 ok-script 的鸣潮自动化工具。它证明 ok-script 能做真实产品级游戏助手，而不只是 demo。

值得学：

- 产品文档、FAQ、后台模式说明、快捷命令、打包升级都很完整。
- 任务代码大量使用统一的 `BaseTask` 操作 API，业务逻辑不直接关心底层输入实现。
- 游戏语言、分辨率、OCR、模板特征、任务调度被放进可配置体系。

对 YHFish 的启发：

- 需要把“能跑”升级成“能诊断、能自测、能解释失败”。
- 真机 smoke 不应该只靠人工逐节点点，应该有任务级报告。

### MaaFramework

源码：<https://github.com/MaaXYZ/MaaFramework>  
文档：<https://docs.maa.plus/zh-cn/>

定位：C++20 黑盒图像识别自动化框架，支持 Windows、Linux、macOS、Android。它的最大价值是控制器和 pipeline 协议。

值得学：

- **Controller 是一等对象**。ADB、Win32、Android Native、macOS、Linux、Gamepad、Replay、Record 都是 controller。
- **截图和输入方法可枚举、可组合、可测速或可降级**。
- Win32 明确承认没有通用输入方式：Seize、SendMessage、PostMessage、LegacyEvent、WithCursorPos、WithWindowPos 等要并存。
- ADB 也不是单一路径：AdbShell、Minitouch、Maatouch、EmulatorExtras；截图有 Encode、Raw、Minicap、模拟器私有通道。
- Pipeline 区分 recognition、action、next、on_error、target、roi、anchor、wait_freezes、trace detail。
- 提供 Debug/Record/Replay controller，这对“不一个个手动验证节点”非常关键。

不适合直接照搬：

- C++ native 分发、绑定、升级成本高。
- LGPL-3.0 要认真做合规边界。
- Maa pipeline 不应该替代 YHFish 的节点图；YHFish 的核心资产是可视化节点产品。

推荐取舍：

- 学它的 **Controller/Capability/Trace** 边界。
- 不直接把 Maa pipeline 作为 YHFish 图执行内核。
- 未来可把 Maa 作为可选 controller provider，而不是核心依赖。

### Airtest / Poco

源码：<https://github.com/AirtestProject/Airtest>  
文档：<https://airtest.readthedocs.io/en/latest/>

定位：跨平台 UI 自动化框架，支持 Android、Windows、iOS，图像识别优先；Poco 补 UI 层级访问。

值得学：

- `connect_device("Android:///...")`、`connect_device("Windows:///?title_re=...")` 这种设备 URI 很适合 YHFish。
- `Device` 抽象统一 `snapshot/touch/swipe/keyevent/text/start_app/stop_app`。
- 操作 API 自动记录日志和截图，可生成报告。
- Android 截图/输入方法分层：adbcap、javacap、minicap、minitouch、maxtouch、Yosemite IME。
- Windows 下也处理 DPI、窗口 rect、相对坐标、截图 fallback。

不适合直接照搬：

- Airtest 是 Python 脚本和测试框架，不是节点执行引擎。
- Windows 后台能力不如 MAA 的 Win32 控制层细。
- 对浏览器和企业 RPA 组件不是强项。

推荐取舍：

- 学它的 `Device URI`、报告、断言节点、录制回放。
- 不把 Python runtime 引入主执行链。

### 影刀 / UiBot / Power Automate

影刀：<https://www.yingdao.com/>  
UiBot：<https://docs.uibot.com.cn/>  
Power Automate Desktop UI automation：<https://learn.microsoft.com/en-us/power-automate/desktop-flows/actions-reference/uiautomation>

定位：商业 RPA 产品，重心是办公、浏览器、企业系统、数据处理、调度、审计、组件市场。

值得学：

- 录制器、组件库、变量表、数据表、凭据、调度、日志审计、异常处理。
- 参数面板一致性非常重要；同类节点的输入命名和帮助文本必须统一。
- 商业 RPA 都把“运行报告”和“失败定位”当核心产品能力。

对 YHFish 的限制：

- 它们不是为游戏/模拟器/后台 Win32 raw input 设计的。
- 商业产品文档无法给我们底层实现细节。
- 元素自动化对游戏、AE 这类复杂 native UI、浏览器 canvas、模拟器画面帮助有限。

推荐取舍：

- 学产品层：录制、参数一致性、运行日志、异常恢复、调度。
- 底层控制不要照商业 RPA 的元素点击模型。

### Appium

官网：<https://appium.io/docs/en/latest/>

定位：移动端 UI 自动化标准生态，靠 driver 访问系统 UI 树。

值得学：

- driver/plugin 架构。
- Android/iOS app 的 UI 层级自动化能力。
- 适合普通 App，不适合游戏画面和模拟器黑盒图像。

推荐取舍：

- Android 普通 App 自动化可以借鉴 Appium 的 capability/session 设计。
- 游戏/模拟器仍以 ADB 截图 + 输入 + 视觉识别为主。

### SikuliX / TagUI / Robot Framework

SikuliX：<https://sikulix.github.io/docs/>  
TagUI：<https://github.com/aisingapore/TagUI>  
Robot Framework：<https://robotframework.org/>

定位：图像脚本、RPA DSL、测试自动化框架。

值得学：

- 简单 DSL、测试报告、关键字抽象。
- 图像点击和断言的用户心智简单。

不适合作为主参考：

- 底层多目标控制、Win32 后台输入、Android 模拟器优化不如 Maa/Airtest/ok-script。
- YHFish 已经有可视化节点，不需要再引入文本 DSL 作为主模型。

## 对 YHFish 的架构启发

### 1. Window 不是足够稳定的核心抽象

当前 `Window` 能解决一部分 Win32 问题，但它不能覆盖：

- Android 设备 / 模拟器。
- 浏览器页面 / CDP target。
- AE 主窗口和新建合成弹窗这种同进程多窗口目标。
- 截图目标和输入目标不一致的情况。
- Replay / Mock / Record 这种非真实窗口目标。

后续应该引入 `Target`，`Window` 只是 `TargetKind=win32-window` 的一种。

### 2. 输入后端不能是容器全局常量

Win32、浏览器、Android 的输入方式差异很大；同一个软件内不同动作也可能要不同策略。

例子：

- AE `Ctrl+N` 需要前台 keyboard injection。
- AE 弹窗点击需要 target 切到新窗口。
- 浏览器点击最好走 CDP 或 DOM/Accessibility，不一定走鼠标。
- Android 点击先走 ADB shell，后续可换 minitouch/maatouch。

后续应由 `ActionRouter` 按 `Target + Action + Capability + Policy` 选择后端。

### 3. 坐标系必须显式建模

需要至少区分：

- `screen`：桌面绝对坐标。
- `window-client`：Win32 客户区坐标。
- `capture-frame`：截图帧坐标。
- `device`：Android 设备触控坐标。
- `browser-viewport`：浏览器 viewport 坐标。
- `normalized`：0-1 相对坐标。

任何截图取点和点击都必须带 `CoordinateSpace`，不能只传裸 `x/y`。

### 4. Trace 是核心功能，不是日志附属品

每个节点执行应能保存：

- 执行前 target 信息。
- 执行前截图。
- 识别结果和置信度。
- 坐标转换链。
- 输入后端和实际参数。
- 执行后截图。
- 成功/失败原因。

这能把“一个个手动验证节点”变成“跑 smoke + 看报告”。

## 总体推荐

短期不要重写语言，不要引入 Maa 作为核心，不要把 Android 做成旁路。先升级内部模型：

```text
Graph Node
  -> Action Request
  -> Target Resolver
  -> Controller Capability Probe
  -> Action Router
  -> Screenshot/Input Backend
  -> Trace Report
```

YHFish 的长期优势应该是：**可视化节点产品 + 多目标自动化内核 + 可解释运行报告**。

