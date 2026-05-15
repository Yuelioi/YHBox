# piano — 自动弹琴

解析 MIDI 文件 → 时序 PostMessage 按键到异环钢琴 UI。

## 工作原理

1. **MIDI 解析**：用 [gitlab.com/gomidi/midi/v2/smf](https://gitlab.com/gomidi/midi/v2/smf) 解析 SMF 文件 → 提取每个 track 的 note events
2. **选轨**（[track_pick.go](../../tools/piano/track_pick.go)）：
   - `-1` 智能选轨：按音符数 + 音域选最像主旋律的 track
   - `-2` 全部 track 合并
   - `>=0` 手动指定 track index
3. **音域映射**：异环钢琴 36 键 (含半音) / 21 键 (仅自然音)，把 MIDI note 映射到 0-35 / 0-20
4. **自动 transpose**（可选）：整曲八度平移让最多音符落进游戏可弹范围
5. **MelodyOnly**（可选）：同时音只留最高音（去伴奏）
6. **OctaveOffset**：在自动 transpose 基础上手动 ±2 个八度
7. **时序按键**：按 MIDI 时间戳 + 当前 tempo，goroutine 在到点时 `input.Tap(key)` 发到游戏

## 内置曲库 vs 用户文件

- **内置**：`tools/piano/midi/` 下的 `.mid` 文件 embed 进 exe，[library.go](../../tools/piano/library.go) 用 `fs.WalkDir` 枚举
- **用户文件**：通过 wails3 文件对话框选磁盘上的任意 `.mid`，settings 记 `lastMidiPath` 下次启动自动加载

为啥两条路径：内置曲库给"傻瓜双击就能用"场景；用户文件给"自己找 MIDI"场景。

## 关键设置

| Settings.Piano | 含义 |
|---|---|
| `Mode` | 0=36 键 / 1=21 键 |
| `TrackPick` | -1=智能 / -2=全部 / >=0=指定 track |
| `AutoTranspose` | 整曲自动八度对齐 |
| `OctaveOffset` | 手动八度微调 (-2..+2) |
| `MelodyOnly` | 同时音只留最高 |
| `LastMidiPath` | 最近用的外部 MIDI 路径 |

## locale 无关

piano 是纯 MIDI → 按键，**完全不依赖视觉/locale**。没有 templates 也没有 ROI 坐标。`tools/piano/midi/` 目录直接 embed，不分 locale。所以 piano 没有 `configs/<locale>/` 子目录。

## 关键文件

- [piano.go](../../tools/piano/piano.go) — Run 主循环（按时间戳 trigger）
- [library.go](../../tools/piano/library.go) — 内置曲库枚举 + LoadEmbeddedMIDI / LoadFileMIDI
- [smf.go](../../tools/piano/smf.go) — MIDI 解析 + parseSMF
- [track_pick.go](../../tools/piano/track_pick.go) — 智能选轨算法
- [transpose.go](../../tools/piano/transpose.go) — 自动八度对齐
- [keymap.go](../../tools/piano/keymap.go) — MIDI note → 游戏键位映射

## 已知坑

- MIDI 文件 tempo 改变（meta event TempoMicrosecondsPerQuarter）必须正确处理，否则后半段速度全错
- 部分 MIDI 用 PPQ + 多 tempo event，全转 absolute time 比相对时间稳
- AutoTranspose 对复杂多 track MIDI 有时挑错八度，加 OctaveOffset 手调
- 智能选轨偶尔挑到伴奏轨（音符多但不是主旋律），改 TrackPick 为具体 track 编号
