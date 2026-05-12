# YHBox — 异环工具箱

[![GitHub](https://img.shields.io/github/v/release/Yuelioi/YHBox)](https://github.com/Yuelioi/YHBox/releases)

Windows 工具，为《异环 / Neverness to Everness》提供后台自动化。**真后台**：不抢前台焦点、不动鼠标光标，挂机时可以正常用电脑写代码、看视频、刷网页。

单 exe，无外部依赖，~2MB。

|        自动钓鱼        |         自动弹琴         |
| :---------------------: | :----------------------: |
| ![钓鱼](preview/fish.png) | ![弹琴](preview/piano.png) |

## 功能

- **自动钓鱼（fish）**：全自动抛竿 / 等待上钩 / 溜鱼 / 收杆 / 购买鱼饵 / 填充鱼饵 / 售卖鱼获 / 处理结算；带运行时统计（时长 / 总数 / 普通·紫色·金色按溜鱼时长分类）
- **锤子连点（cook）**：检测烹饪界面的锤子图标并自动点击（目前只支持一直点，后续再补充自动重复当前关卡, 有自动钓鱼消耗都市体力 cook模式我感觉连点通关成就就够了）
- **自动弹琴（piano）**：解析 MIDI 自动演奏；内置曲库 + 任意 `.mid` 文件；智能选轨 / 自动八度对齐 / 只弹主旋律 / 进度条拖动；支持游戏 36 键（含半音）/ 21 键（仅自然音）两种键位

## 快速开始

下载 `YHBox.exe`，双击运行，UAC 弹窗点"是"。

界面是带 tab 的 GUI 窗口：

- **自动钓鱼 / 锤子连点 / 弹琴**：在对应 tab 点 `开始` / `暂停` / `停止`
- **设置**：日志显示开关 / 自动滚动 / 统计区显示
- **帮助 / 关于**

设置 / 窗口位置自动保存到同目录的 `settings.json`。

## 使用前确认

1. 游戏窗口不能最小化
2. 游戏分辨率 16:9 等比（720p / 1080p 实测；1440p / 4K 可能也支持，未测试；21:9 ultrawide 暂不支持）。推荐 **1920×1080 无边框窗口化**
3. 钓鱼前角色需到达钓鱼点，屏幕右下角出现 **[F] 抛竿** 提示
4. 锤子连点前需打开烹饪界面
5. 弹琴前需在游戏内打开钢琴演奏界面，并按钢琴键位选好 36 键 / 21 键模式

### ⚠️ 抢占鼠标的场景

有些场景只能用鼠标点击，会干扰正常使用：

- 店长模式锤子连点
- 钓鱼的「开始钓鱼」/ 购买鱼饵 / 切换鱼饵 / 出售鱼饵

钓鱼大部分场景不受影响。

---

## 项目结构

```text
.
├── assets/                  识别模板 PNG + 内置 MIDI（embed 进 exe）
├── assets.go                embed 入口
├── cmd/yhbox/
│   ├── main.go              入口：EnsureAdmin → gui.Run(version)
│   ├── gui/                 walk MainWindow + tabs + 彩色日志 (RichEdit50W) + 统计
│   └── winres/              图标 / 版本信息 / DPI manifest
├── pkg/
│   ├── capture/             PrintWindow 后台截图（全帧 / FrameROI）
│   ├── input/               WM_ACTIVATE + PostMessage 后台输入
│   ├── log/                 多 sink 日志器（GUI + 文件）
│   ├── platform/            UAC 提权
│   ├── runctl/              GUI ↔ bot 的 Pause / Resume / Stop 控制接口
│   ├── vision/              模板匹配 + 多尺度金字塔
│   └── winutil/             游戏窗口查找
└── tools/
    ├── fish/                钓鱼状态机 + 检测器 + 耐力条分析 + 控制器
    ├── cook/                锤子连点
    └── piano/               MIDI 解析 + 智能选轨 + 自动 transpose + 时序按键
```

> 维护者 / 二次开发：实现细节、踩坑记录、设计权衡见 **[INTERNALS.md](INTERNALS.md)**；钓鱼模块完整状态机 / 时序见 **[fish.md](fish.md)**。

## 从源码构建

```powershell
go build -ldflags="-s -w -H=windowsgui" -o YHBox.exe ./cmd/yhbox
```

`-H=windowsgui` 必加，否则双击启动会带 cmd 黑框。需要 Go 1.21+，Windows 环境。

或者用 Taskfile：

```powershell
task build       # 带图标和版本信息
task pack        # 构建并用 UPX 压缩
```

---

## 许可

[LICENSE](LICENSE)
