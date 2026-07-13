---
kind: trap
summary: "改存储约定/config schema 前必须 exhaustive grep 全消费者，外部 reviewer 结构上 catch 不到漏掉的读取点"
activation: symptom
read_when: "写\"统一存储约定 / 改 config schema / 改 pin 值读写法\"类 spec 前; impl 第一步就撞\"还有一堆地方在直接读这个 key\"; 评估改 config key 的真实影响面"
---
# ⚠ Incident — 统一存储约定时漏审消费者, scope 翻倍
**Date**: 2026-05-29
**Context**: input-editing-unification (pin 值 顶层 config ↔ config.literal 双轨统一)

## Symptom

写了一份"完整" spec (含正源决定 + 判别规则 + 影响文件), 经 3 个 AI reviewer (claude/gpt/deepseek) 全审通过、决策点全拍板, 开始 impl。**impl 第一步 (改 runner setup) 就撞墙**: `validator.go` 遍地直接读顶层 config 取 pin 值, 且不像 runtime 那样 literal-first merge。spec + 3 reviewer **全漏了 validator 这个主要消费者**。继续挖: rewriter / listener / scanner / recording / configtypes / 一堆 validator_*.go 全在用小写或顶层 key 读 pin 值 (codebase-wide case-drift)。真实工作量从"动一点 runner setup"翻成"全 Go 侧 ~14 文件消费者审计 + case-drift 对齐"。

## Root cause

**假设**: spec 列的"影响文件"(editor + runtime dispatch/setup) 就是全部消费者。
**实际**: 改一个**存储约定**会触及**每一个读该 config key 的地方** — validator (静态校验)、rewriter (库导入改键)、listener (OnEvent)、scanner (依赖提取)、recording (窗口解析)。这些不在"明显"路径上, 但全是消费者。

外部 reviewer **结构上无法**catch 漏掉的消费者: 他们只能 vet spec 内部一致性, 不会去 grep 整个 codebase 找你没列的读取点。多 reviewer 全 approve ≠ 消费者列全了。(呼应头号铁律: review 不替代源码 vet。)

## Remediation / 下次怎么做

**写"统一存储约定 / 改 config schema"类 spec 前, 先跑一次 exhaustive 消费者审计** (grep 全部 `.Config[` / 各读取 helper `configString`/`stringFromConfig`/`xxxFromConfig` / `["literal"]`), 把每个读写点列全 + 分类 (pin值→走 accessor / 元数据→留), **再**定 scope 和影响文件。审计可以派 agent 并行扫, 但转换自己动手。

**配套手法 (本次验证有效)**:
- 引入**单一 accessor** (`PinValue` literal优先+顶层fallback, 镜像 runtime `newInputs` 优先级) 让所有消费者读法一致 — "validator 只读顶层"本身就是跟 runtime 不一致的既有 bug, 统一是修 bug 不是加 shim。
- accessor 的**顶层 fallback** 让既有数据无需迁移即可正确读 → 迁移脚本从"必须"降级为"可选清理", 大幅降风险。

## 关联

- 历史材料: cold archive `2026-05-29-input-editing-unification` (§3.10 记录 reopen + §8 实况)
- 同源既有 bug (本次顺带暴露, 未修): Switch FE/BE schema 漂移 / screen-pick case-drift / MouseCalibration 旧小写 counts360。

## [Case 2] 2026-06-10 — eval cache 漏第二个 EvaluatePureData 调用点 (random-nodes C1)

同根因变体: random-nodes spec 的"已验证源码事实"写"纯数据求值只有 `evalDataSource` 一个路径", 据此把 per-dispatch 缓存 gate 只加在那里。实际 `EvaluatePureData` 有**两个**生产调用点 — 漏了 `dispatch_v5.go::resolveDataPinV5` 的直连分支(恰是 exec 节点 data-in 的**主路径**), 缓存被完全绕过, 核心承诺(同 dispatch 多路径同随机值)在真实派发里不成立。spec 过了 2 轮外部 AI 审都没发现(reviewer 不会 grep 调用点, 同 Root cause)。impl 阶段质量审用 overlay 探针实测才抓到, 修法 = 两路并入 `evalPureDataCached` 单一 gate (commit 50de637)。

**教训重申**: 下"X 是唯一路径/唯一调用点"的结论前, 必须 `grep 该函数名` 列全调用点 — 这次漏的不是"隐蔽消费者", 而是同文件邻函数里的直连分支。
