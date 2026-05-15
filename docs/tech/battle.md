# battle — 一键切队伍

全局热键 (Win32 `RegisterHotKey`) 触发的一次性流程：按 `Ctrl+Shift+1~6` 切上阵队伍 1-6。

## 工作原理

1. **热键注册**：`Settings.UI.Battle.HotkeyEnabled=true` 时 [hotkey.go](../../cmd/yhbox/services/hotkey.go) 在锁定 OS 线程上 `RegisterHotKey + PeekMessage` 循环监听 `WM_HOTKEY`
2. **触发**：用户按热键 → onHotkeyFired(id) → 起一个 goroutine 跑 `SwitchTeam(teamIdx)`
3. **SwitchTeam 流程**（[switch.go](../../tools/battle/switch.go)）：
   - 检测游戏窗口是否前台 + 主界面（无弹窗）
   - 按 `L` 打开编队 UI
   - 模板匹配找到对应 team tag 位置
   - Click → 等切换成功 toast（`team_switch_success` slot）
   - 按 ESC 关编队 UI

## 为什么用全局热键

切队伍是即时操作，用户在玩游戏想立刻换队，不该切到 YHBox 窗口点按钮。全局热键 = 在游戏内按下立即响应。

## 关键文件

- [battle_service.go](../../cmd/yhbox/services/battle_service.go) — Enable/Disable + AutoStartFromSettings + cancelRunning
- [hotkey.go](../../cmd/yhbox/services/hotkey.go) — Win32 RegisterHotKey 封装（锁线程 PeekMessage 循环）
- [switch.go](../../tools/battle/switch.go) — SwitchTeam 流程
- [configs/zh/templates.toml](../../tools/battle/configs/zh/templates.toml) — 8 PNG (team_tag / team_enabled / team_select / team_switch_success × 2 分辨率)

## 修饰键 5 选 1

`Settings.UI.Battle.HotkeyMods` 支持：`Ctrl` / `Ctrl+Shift` / `Ctrl+Alt` / `Shift+Alt` / `Ctrl+Shift+Alt`。

5 个组合够避开常见占用（截图工具 / 输入法 / 录屏软件）。改 mod 后会自动 UnregisterHotKey + 重新 RegisterHotKey。

## 关闭程序自动注销

app.Shutdown 会自动 `battleService.Disable()` 反注册所有 hotkey。Win32 进程退出时 OS 也会自动清理该进程注册的 hotkey，所以即使崩溃也不会泄漏。

## 已知坑

- 游戏不在前台 / 弹窗占用 L 键 → 流程中止（按 L 打不开编队 UI）。UI 显示"未检测到主界面"
- 热键被其它应用占用（如截图工具）→ RegisterHotKey 返失败。换 mod 组合解决
