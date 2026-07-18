---
kind: trap
summary: "录制必须把 recorder event、session state、canonicalization、codec、asset 与 playback 当成一个契约；手写合法 fixture 会掩盖真实事件不变量。"
activation: symptom
read_when: "修改录制事件/状态/finalize/codec/clip asset/playback，或出现 HUD 卡住、保存 ordering invalid、录制成功但回放失败时"
recheck_when: "Recording Session、inputclip event schema、HUD state merge、asset commit 或 playback adapter 改变后"
---
# Recording contract cascade

录制不是 recorder、HUD、codec、Asset Store 和 playback 五个独立功能。正确链路是：

```text
start + state snapshot/event stream
→ raw native events
→ pause-aware canonicalization
→ finalize
→ codec validate/encode
→ asset/blob atomic commit
→ reload/decode
→ admitted playback
```

已确认的假绿模式：codec 要求首事件 `TUs == 0` 且 `(TUs, Seq)` 严格有序；单测手写的事件已满足不变量，但真实 recorder 未归零，最终保存失败。测试必须把 recorder 产出的原始事件直接送入 finalize/codec，而不是重新手工构造“正确数据”。

HUD/页面必须在订阅事件前后读取 monotonic session snapshot；只监听未来事件会错过 start 状态并永久停在“准备中”。RPC transport 失败要 rethrow 原始结构化错误，不能返回 `undefined` 再制造 `recording.finalize: invalid result`。

simple/precise 是用户可见且写入 carrier 的录制策略：simple 过滤 incidental trajectory，precise 保留轨迹和 timing。不能用“出现任意 mouse move”启发式静默改变产物类型。录制应优先使用 native event clock，在唯一 canonicalization 边界重排、归零和补 release；不能在 hook drain、页面和 codec 各修一遍时序。

相对鼠标校准属于 exact installed target profile，不属于独立全局 Settings snapshot。Session 从 Start 到 Stop/Cancel 必须持有同一 target generation lease，否则设置热更新会让一次录制跨两个 identity/calibration/backend。取消、暂停、继续、F11/F12、held key/button cleanup 和 pending/finalize/discard 都属于同一 Session 生命周期。

完成证据必须包含真实输入 → 保存 clip → 资源库可见 → workflow picker → playback；只通过 codec、store、页面或 hook 单测不能称录制能力完成。
