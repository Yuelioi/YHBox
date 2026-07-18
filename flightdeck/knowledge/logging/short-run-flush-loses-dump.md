---
kind: trap
summary: "历史 3.0 Container LogMerger 短图 flush 陷阱；现行 3.1 Run journal/debug 不使用该节点 dump 路径。"
activation: symptom
read_when: "仅在审查 3.0 Container node-dump/LogMerger 旧行为或归档问题时"
recheck_when: "LogMerger flush/tick/finalize 逻辑改 / logMergerFlushInterval 改 / 前端 appendNodeDump 的 (nodeId,lineKey,frozen) 幂等键改"
---
# ⚠ 短图节点日志在前端丢失 — LogMerger.finalizeLocked 旧实现只写文件不 emit

> 历史知识：旧 Container node-dump 事件链已删除；3.1 节点事实以 RunRecord/NodeAttempt/AdapterAction 为准。
**Date**: 2026-06-25 (detect-click 真机 smoke 期间发现: 节点全勾 logEnabled, 运行经常没日志; container da4755f5 短图 Win32WindowTarget→BringForeground→InputText→Stop)

## 根因
节点级「启用日志」(`logEnabled`) 执行时 `emitDump` 发 `container:node-dump` → app.go 喂 `LogMerger.Add` 合并 → 合并后内部发 `container:node-dump-batch`，再由 `LogSink` 归一化进统一 `log:batch` 前端流；文件仍独立写入。

普通 (非 error) dump 行进 `Add` 后**不立即 emit**, 只建/更新一个 `dirty` 段。它的前端 emit **唯一**路径是 `tick()` (每 `logMergerFlushInterval`=250ms 扫 dirty 段批量发)。而容器结束信号 → `FlushContainer` → `finalizeLocked`, **旧实现只 `writeFile` 不 emit**。

⇒ 容器在第一个 250ms tick 之前跑完 (短图就这样), dirty 段还没被 tick emit, FlushContainer 直接 finalize 写文件但**从不 emit** → 前端面板零日志, 但**日志文件里有**。跑得慢 / 跨过 250ms tick → 才被 tick emit, 偶尔「有日志」。这就是「经常没日志」的间歇性。

同 bug 的另一面: `Add` 里同节点换 lineKey 时 `finalizeLocked(旧段)` 也不 emit → 同节点快速变输出 (<250ms) 时旧 lineKey 段在面板丢失。

测试缺口: 旧 `logmerger_test.go` 只断言 `files` (写文件) 从不断言 emit, 所以这个「不 emit」从没被测到。

## 修复 (已做)
根因 = finalize 路径不 emit。一处修干净: `finalizeLocked` 收尾时 `emitOneLocked(..., final=true)` 定版 + writeFile + 删除; `tick` 的 idle 分支改为只调 `finalizeLocked` (不再自己 append 进 batch, 免重复)。dirty(活跃) 段仍由 tick 批量 emit `final=false` 中间态。

前端 `appendNodeDump` (stores/log.ts) 按 `(nodeId, lineKey, !frozen)` 幂等更新: 段先 emit 中间态、后 emit final 只更新同一行并置 `frozen`, **不重复行**; 段 frozen 后下一批次同 key 另起新行。所以「finalize 总 emit final」对前端安全。

新增回归测试 `TestLogMerger_FlushEmitsUnflushedSegments`: Add 两节点后不等 tick 立刻 Flush, 断言必 emit 2 段且 final=true (修复前 emit 0)。
