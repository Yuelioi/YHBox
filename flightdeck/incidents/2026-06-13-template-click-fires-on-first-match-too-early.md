---
name: template-click-fires-on-first-match-too-early
description: ClickTemplate/WaitTemplate「点了不生效/点空」根因 —— WaitMatch 在模板**首次越过阈值那一刻**就 return, 节点立刻动作(点击/放行), 且**只动作一次不重试**。游戏转场/加载时模板"刚冒出来"还在动画/未就位/未可点 → 这一下太早 → 落空。诊断关键: 在节点前垫个 Sleep 就好(跟动画时长无关, 动画≤200ms 也要等十几秒, 因为模板要等加载完才出现, 出现即被点)。修法: 加 SettleMs(命中后稳定延迟 + 用新鲜帧重定位再动作), 抽 settleAfterMatch 共用。附带订正: 轮询是 100ms(visionWaitPollMs)不是注释里的 250ms; WaitMatch 确实每轮抓新帧(缓存 100ms TTL 每轮过期)—— "没一直检测"是误解, 真因是"认第一次命中就动作"
when_to_read: 撞 ClickTemplate/WaitTemplate「检测到了/走了 Done 但游戏没反应」「点空」「点了不生效」; 怀疑模板在转场帧上误命中或点早了; 设计任何「检测到模板→立刻点击/动作」类节点前; 想搞清 WaitMatch 轮询/帧缓存/超时的真实机制
applies_to: [detect, template-match, click-template, wait-template, WaitMatch, settle, SettleMs, vision, frame-cache, visionWaitPollMs, transition-timing, internal/nodes/detect/click_template.go, internal/nodes/detect/wait_template.go, internal/nodes/detect/template_common.go, internal/services/container/runtime/node_services.go]
last_updated: 2026-06-13
status: active
---

# 模板点击「认第一次命中就点 → 转场时太早 → 点空」+ SettleMs 修法

**Date**: 2026-06-13（用户报 ClickTemplate 自动跑「点了不生效」, 手动单跑/前面垫 Sleep 就好）

相关前案: [[slate-click-up-coords-and-hold-lifecycle]]、[[node-timed-input-loses-backend-activate]]（都是输入落点/时序; 本案是**检测时序**, 不是落点也不是激活)。

## 根因（用户实测钉死, 排除了"未就绪/未激活"）

`ClickTemplate.Run` / `WaitTemplate.Run` 调 `ctx.Vision().WaitMatch(...)`，它**一命中就立刻 return**（首个越过 Threshold 的帧），节点**立刻动作**（ClickTemplate 点击 / WaitTemplate 放行下游），而且**只动作一次、不重试到成功**。

游戏**转场/加载**时，目标模板往往"画面上先冒出来一下"（像素够像、越过 0.85），但这一刻它**还在淡入/位移 / UI 还没就绪不可点**。WaitMatch 认这第一次命中 → 立刻点 → 落空 → fire Done 走人（后面画面稳了它早走了，不会回来重点）。

**为什么「没一直检测」是误解**：WaitMatch 是真·持续轮询新帧的 —— `visionWaitPollMs=100`（注: click_template.go 顶注释旧写 250ms 是错的，本轮已订正），帧缓存 `frameCacheTTL=100ms`，轮询每轮 sleep 100ms 正好让缓存过期 → 每轮重抓新帧。所以它**一直在看最新画面**；问题不在"检测频率"，在"**认第一次命中就动作**"。

**为什么不是"未激活"**：`ClickButton`（input.go:281）每次点击开头都无条件 `FakeActivate`，激活每次都新鲜，与本案无关（这条排掉过, 见对话）。

**为什么不是"等动画就绪"那么简单**：用户实测动画 ≤200ms，但要等十几秒才好 —— 因为模板本身要等**加载完**才出现（~12s），出现即被点。所以 Sleep 要够长是为了"等模板出现并稳定后再开始检测"，不是等一个 200ms 动画。

## 修法（本轮，已加单测 + 真机验证）

给 `ClickTemplate` + `WaitTemplate` 加 **`SettleMs`（命中后延迟，默认 0 = 旧行为）**：命中后等 `SettleMs` → **用新鲜帧重定位一次**（`WaitMatch(...,0)` 单帧）→ 才点击/放行。抽成 `template_common.go::settleAfterMatch` 两节点共用。可取消（settle 期间 graph stop → 返 `ctx.Err()`）。

比"前面单独拖个 Sleep"强两点：① **按实际命中计时**（模板啥时出现就从那时等，不盲等固定时长）；② **等完重读坐标**（元素还在动也跟得上；重定位丢了退回原坐标）。

**不适用的节点**：CheckTemplate / DetectColor / ROIColorScan / DetectColorBlobs 是**单帧瞬时**检测（没有"等待"过程），SettleMs 无意义 —— 要延迟就拖 Sleep。

## 留尾 / 旁注

- 本轮顺手给 dump 日志加了 `took=<耗时>`（节点 Run 实际耗时，`engine.go` 里计时）—— 这类"等了多久才命中"的问题，日志直接 `took=12.3s` 一眼看出，是排这类 bug 的利器。出口名（→Done/→Timeout）= 命中/超时、`err=` = 失败，本就有。
- SettleMs 是**命中后**等；不能解决"模板根本没出现/出现晚"（那是 TimeoutMs 的事）。两者配合：TimeoutMs 等模板出现，SettleMs 等它出现后稳定。
