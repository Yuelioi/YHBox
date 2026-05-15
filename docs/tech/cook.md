# cook — 锤子连点

烹饪界面里锤子位置固定 → 直接 click 固定坐标，无模板匹配。

## 工作原理

1. 启动期读 [configs/zh/config.yaml](../../tools/cook/configs/zh/config.yaml) 拿当前分辨率的 hammer 坐标（1080p `(103, 457)`、720p `(67, 303)`，等比缩放）
2. 启动 keepalive goroutine 每 500ms `FakeActivate` 保游戏 IsActive
3. 主循环：`input.Click(hwnd, hammerX, hammerY, ...)` → sleep `Settings.Cook.IntervalMs` → 循环

没有抓帧、没有 NCC、没有 miss 兜底。

## 为什么不用模板匹配

锤子图标在烹饪界面是**固定坐标**——抓帧 + NCC 每次 ~8ms 跑等于浪费 CPU 找一个永远在同一位置的东西。

v1 用模板匹配，理由是"允许用户切到别的界面 bot 自动停"。实际：
- 真切到别的界面（如背包/地图），那个坐标多半也有按钮 → 还是会乱点
- 解决"防呆"的正确做法是 GUI 提示用户"在烹饪界面再启动"，不是 bot 内部检测

v2 砍掉模板匹配后 CPU 占用降一档，代码从 ~210 行降到 ~80 行。

## 前台限制

cook 必须**前台运行**：
- `input.Click` 走 `SetCursorPos` 移真光标到 hammer 位置
- 失焦时 `SetCursorPos` 可能把光标移到游戏窗口外（游戏不在前台屏幕区域）

这跟 fish / rhythm / battle 不同。后三个走纯键盘 PostMessage，能后台。cook 结构上没法纯后台（除非游戏支持 PostMessage WM_LBUTTONDOWN 不依赖光标位置，实测不支持）。

## 加新分辨率

只要在 [config.yaml](../../tools/cook/configs/zh/config.yaml) 加一条 entry：

```yaml
matching:
  resolutions:
    "<W>x<H>":
      hammer: [<center_x>, <center_y>]
```

量法：游戏切到目标分辨率，截图，量锤子图标中心像素坐标。

## 关键文件

- [cook.go](../../tools/cook/cook.go) — Run() 主循环（~80 行）
- [config.go](../../tools/cook/config.go) — Config struct + HammerPoint + PickHammer
- [cook_loader.go](../../tools/cook/cook_loader.go) — LoadConfig + yaml 反序列化
- [configs/zh/config.yaml](../../tools/cook/configs/zh/config.yaml) — hammer 坐标

## 已知坑

- Click 的 `WM_MOUSEMOVE` 不要 PostMessage（cook 漏点从 30% 暴涨到 70%）—— 详见 [architecture.md §3.3](architecture.md#33-click-时序down-up-必须紧贴)
- 用户同时动鼠标 → `SetCursorPos` 跟用户操作打架，漏 ~15%。前台运行期间不要碰鼠标
- 切到非烹饪界面再启动 → 那个坐标可能有别的按钮，会乱点。启动前确认在烹饪界面
