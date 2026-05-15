# rhythm — 超强音自动音游

识别 4 轨命中圈音符 → 异步按 D/F/J/K。1080p / 720p 实测 100% 命中。

## 算法来源

借鉴自 [BnanZ0/ok-nte](https://github.com/BnanZ0/ok-nte)（基于 [ok-script](https://github.com/ok-oldking/ok-script) 框架的异环音游自动化）。所有数值参数（亮度阈值 100 / dark_ratio 0.06 / retrigger 85ms / KeyHold 5ms / 4 轨命中点比例）沿用 ok-nte 调好的值。本项目做的是 Python numpy 实现 → Go 移植 + 集成进 YHBox bot 框架。

### 历史演进：HSV → 亮度

**早期方案（HSV 颜色匹配，95% 命中）**：4 轨各自维护 cyan/orange/red/purple 的 HSV 阈值，落进阈值的像素加权累加（中心 ×2 / 边缘 ×1），cnt 超 HighThresh 触发，cnt < LowThresh 回 Idle。死区抗抖。

为什么 95% 卡点：HSV 阈值边界敏感。音符进 ROI 边缘时颜色未饱和、anti-aliasing 让 cnt 卡边界，偶尔卡死区漏触发。

**当前方案（亮度检测，100% 命中）**：每像素 `(R+G+B)/3 < 100` 算暗，整 ROI 内暗像素占比 ≥ 0.06 触发。背景 ≈ 245、音符 ≈ 28，亮度差大，对 AA / 饱和度不敏感。

副产品改进：
1. **ROI 从 68×85 缩到 11×21**（小 25 倍）：音符进 ROI 即触发，时间精度高
2. **同步 KeyDown → sleep 30ms → KeyUp 改异步 worker + 5ms hold**：detect 主循环不被卡 30ms 漏其它轨

## 检测原理

### DarkRatios（[detect.go](../../tools/rhythm/detect.go)）

每个 ROI 11×21 = 231 像素。遍历每像素：

```go
avg := (int(R) + int(G) + int(B)) / 3
if avg < BrightThresh {  // 100
    dark++
}
total++
```

返回 `dark/total` 是 0.0-1.0 的 ratio。

### TrackState（[state.go](../../tools/rhythm/state.go)）

边沿触发 + retrigger 节流：

```text
prev=false + ratio ≥ 0.06         → fire (上升沿)
prev=true  + ratio ≥ 0.06 + 距上次按下 ≥ 85ms → fire (长按 retrigger)
其它                                → 不触发
```

没有 Idle/Pressed 死区，因为亮度检测稳定，不需要死区抗抖。

### 异步 pressWorker（[rhythm.go](../../tools/rhythm/rhythm.go)）

旧版同步：`detect → KeyDown → sleep 30ms → KeyUp → detect 下一帧`。30ms 阻塞期间错过其它轨的音符。

新版异步：
- detect 主循环 `workers[i].trigger()` 非阻塞 enqueue (buffered cap=4，满则 drop)
- 4 个独立 goroutine 各自消费 channel → KeyDown → sleep 5ms → KeyUp
- 多轨同时触发不互相阻塞

## ROI 标定

4 个 ROI 11×21 居中在命中点。命中点比例（异环 UI 等比缩放）：

| 轨 | 键 | X 比例 | Y 比例 |
|---|---|---|---|
| T1 | D | 0.2301 | 0.7715 |
| T2 | F | 0.4055 | 0.7715 |
| T3 | J | 0.5941 | 0.7715 |
| T4 | K | 0.7715 | 0.7715 |

具体像素 ROI 在 [configs/zh/config.yaml](../../tools/rhythm/configs/zh/config.yaml)：

```yaml
matching:
  resolutions:
    "1920x1080":
      rois:
        - [436, 828, 447, 849]   # T1 居中 (441, 838)
        - [774, 828, 785, 849]   # T2 居中 (779, 838)
        - [1138, 828, 1149, 849] # T3 居中 (1143, 838)
        - [1476, 828, 1487, 849] # T4 居中 (1481, 838)
    "1280x720":
      rois:
        - [290, 549, 301, 570]   # T1 居中 (295, 559)
        ...
```

## locale_alias

rhythm 是纯像素颜色检测，跨语言无视觉差异。[configs/en/manifest.yaml](../../tools/rhythm/configs/en/manifest.yaml) 走 `locale_alias: zh` 复用 zh 配置：

```yaml
version: 1
implemented: true
locale_alias: zh
reason: ""
```

切到 en locale 启动时，`botcore.ResolveLocale` 检测到 alias 自动重定向加载 zh 配置。

## 调参 / debug 工具

### rhythm-calibrate CLI

`go run ./cmd/rhythm-calibrate`：30s 抓帧统计每轨 dark_ratio 分布，输出 min / p50 / p95 / p99 / max 直方图。理想：

- p50 应该 << 0.06（背景多数时间没音符）
- p95 / p99 应该 >> 0.06（音符出现的 tick）
- p95 跟阈值差太近 → 阈值偏高，调低
- p50 接近阈值 → 阈值偏低，调高

### Mock backend 回放

调阈值时不用反复开游戏。流程：

1. 真游戏跑一次 + 设置开"落盘 detect 标注图" toggle，捕获一段 PNG 序列到 `debug/captures/rhythm/`
2. 把帧复制到 `bin/mock-frames/`
3. 设置切到 Mock backend → 重启 exe
4. 启 rhythm，看到的就是回放帧，改阈值立即可见

详见 [development.md §调参](development.md#调参)。

## 常量速查（[constants.go](../../tools/rhythm/constants.go)）

| 常量 | 值 | 来源 / 含义 |
|---|---|---|
| `TickInterval` | 8ms | ~120Hz 主循环节拍 |
| `BrightThresh` | 100 | 像素平均亮度 < 此值算"暗" |
| `DarkRatioThreshold` | 0.06 | ROI 内暗像素占比 ≥ 此值触发 |
| `RetriggerInterval` | 85ms | 长按音符的重发间隔 |
| `KeyHoldMs` | 5ms | KeyDown → KeyUp 间隔 |

改这些走代码 review，不外置 yaml（参数都是 ok-nte 实测出来的，用户调反而坏事）。

## 已知限制

- WGC backend 60Hz vsync 锁定，rhythm 8ms tick 拿到的多数是同一帧（仍然 100% 命中——只要音符进 ROI 那帧能抓到）
- 异常快的谱面理论上可能漏 1~2 个边缘音（音符跨进/出 ROI 在同一帧内）
- 后台运行依赖 `input.FakeActivate` keepalive 让游戏 IMC 保持 IsActive=true 接受按键
