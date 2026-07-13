---
kind: trap
summary: "Sleep 只保证“不早于”目标时间；共享 runner 的调度延迟不能作为 QPC/回放逻辑失败，调度算法应注入 clock/wait 做确定性测试。"
activation: symptom
read_when: "当 QPC/播放/定时测试只在 race、高负载或共享 CI 上超出固定毫秒容差；编写任何依赖 Sleep 后返回上限的测试前。"
---
# ⚠ 实时时序测试把调度延迟误判成时钟错误
2026-07-10，inputclip runtime 在隔离运行和重复 20 次时通过，但与 vet/跨平台 build 并行或 race 高负载时，`Sleep(10ms)` 可能 34ms 后才恢复，导致测试把宿主调度延迟误报为 QPC 不准。

处理原则：

- 单调时钟测试将 QPC elapsed 与同一真实区间的 wall-clock elapsed 比较，不假设 goroutine 必须在固定上限内被调度。
- 调度/裁剪算法注入私有 clock 与 wait 函数，直接断言计算出的 target；不要用真实 wall-clock 验证纯逻辑。
- 真实输入精度属于 Windows integration/performance 检查，不能作为共享 runner 上的确定性单测。
