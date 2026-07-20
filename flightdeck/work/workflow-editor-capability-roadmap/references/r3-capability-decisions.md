# R3 编辑器与节点能力决策

## Evidence boundary

- 旧行为取证固定在 `8316d590dbc8429b783b99982ff30d15e650c59a`，没有把旧 Container runtime 搬回当前树。
- 当前真机 workspace 只包含用户已验证的 `RunStarted → PressKeys(ESC)` 旅程；没有可迁移的 3.0 specialized-vision/EventTick Source artifact。
- 旧 Knowledge 和代码仍证明这些能力的用户意图：Fishing watchdog 使用 `Now − VarLastChange`；`DualColorBarTrack` 面向血条/进度条/QTE/钓鱼条；`ROIColorScan` 做轴向连续色段；`FindColorSignature` 做锚点加偏移颜色组合；`EventTick` 每隔若干毫秒并发 spawn 子 runner。
- 决策口径是用户任务能否在 3.1 唯一 compiler/scheduler、强类型与 capability 边界内完成，不以旧节点名是否原样存在为准。

## Final decisions

| 3.0 能力 | 3.1 决策 | 用户闭环与理由 |
| --- | --- | --- |
| `VarLastChange` | 恢复为 `StateLastChange` | 从 State 面板按变量直接插入；输出强类型 Unix 毫秒，可与 ObserveTime/整数运算组合。 |
| `IncVar` | 恢复为 `StateIncrement` | 从数值 State 的上下文动作插入；在 Run State slot 锁内原子更新，避免旧 read-then-write 竞态。 |
| Stopwatch named table | 替换为显式 Start/Read/Stop 数据流 | Start 输出 `started-at`，Read/Stop 显式消费；不恢复进程全局、按字符串 key 的 ambient table。 |
| `WaitStable` / `WaitChange` | 恢复 | exact target capture + 有界 grid frame-diff + Run cancellation；不恢复共享全局 Vision 快照。 |
| 动态 Switch | 恢复为 typed Switch | 一个 resolved generic type，最多八个可选同类型 case，首个命中或 default；不接收任意动态端口名。 |
| `LoadImage` / `SaveImage` | 恢复为 workspace-safe image I/O | 只读写 Yotta 管理的 workflow files；Image 用 BlobRef，分块、预算、拒绝 path escape/symlink，不恢复任意宿主路径。 |
| `DualColorBarTrack` | 复合替代 | `CaptureWindow → FindColorBlobs(inner/outer) → List/Break/geometry/math` 可封装成 Source-native subgraph；FindColorBlobs 带旧名称搜索别名。旧节点只是这个组合的固定启发式，不再新增第二套视觉服务。 |
| `ROIColorScan` | 复合替代 | `CaptureWindow → FindColorBlobs/AnalyzeColor → list/filter` 提供更一般的二维连通域和颜色统计；旧名称可搜索到替代节点。没有真实 Source 证明必须保留旧一维投影 DTO。 |
| `FindColorSignature` | 复合替代 | 模板资产用于稳定多点图案；颜色任务使用 AnalyzeColor/FindColorBlobs。旧名称可搜索到替代节点；不恢复无资产、逐像素扫描的专用 JSON 签名 DTO。 |
| `MouseCalibration` 节点 | 删除节点、保留产品能力 | 校准是目标机器/输入 profile 属性，不是 workflow 业务数据；设置中心的输入页、F8 HUD、precise recording 校验与 clip calibration metadata 是唯一入口。 |
| `EventTick` | 删除 ambient 节点，提供两条显式替代 | Run 内轮询使用 `Repeat → body → Wait/Delay → continue`，有界且可取消；独立周期任务使用“计划 → 间隔 → Workflow”。Repeat/Delay 带 `EventTick/tick/timer/polling` 搜索别名。旧实现并发 subrunner 共享全局状态，不满足 3.1 scheduler/state/journal 顺序语义。 |

## Architecture consequence

R3 没有给 `InstanceResolver` 增加前端动态端口协议，也没有恢复全局 Stopwatch/Vision/variable service。新增能力深化现有三条边界：Run State 原子更新、exact-target frame observation、workspace file authority。节点 authoring metadata 只提供发现和说明，不改变 machine contract。

## Recheck triggers

- 出现真实 3.0 Source/任务，证明 specialized vision 复合路径无法表达其输出或性能预算。
- scheduler 获得可审计的 structured-concurrency/background event contract；届时可重新评估 EventTick 语义，但不能复用旧共享 subrunner。
- workflow files root、BlobRef authority 或跨平台 target capture contract 改变。

