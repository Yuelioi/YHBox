# Cockpit — YHFish

**Last updated**: 2026-06-10 by 月离 (**加节点路线图 4 批 spec 全部写完 + 多轮外部 AI 审收敛**, 阶段1 另有 TDD plan, 全部零代码未执行。下一对话指令见「下一步」。｜前: 子图 `__missing__` 根治真机验。)
**Active focus**: **加节点路线图**(用户 2026-06-10 起)— 4 阶段 spec **全部写完 + 经多轮外部 AI 审收敛**, **零代码、均未执行**(用户 token 紧, 只设计不跑)。顺序: ①随机 ②数组(提前) ③数学 ④字符串。阶段1 另有完整 TDD plan。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

- 无专项进行中。(modal+HUD 风格统一已完成, 见 Active focus + [ui.md](checklists/ui.md) / [standalone-window-style.md](checklists/standalone-window-style.md)。)

## 下一步

### 🎯 下一个对话该干什么(加节点路线图)

**首选: 执行阶段1 随机** — 它是唯一有完整 plan 的, 且阶段2 的 RandomChoice 依赖它, 必须先落地。
- 跑 [plan](plans/2026-06-10-random-nodes.md)(8 任务 TDD)。推荐 subagent-driven(每任务一 subagent + 任务间复核)或 executing-plans。
- 内容: RandomInt/Float/Bool + 框架 per-dispatch 求值稳定(`IsNonDeterministic` 标志 + dispatchEvalCache)。
- spec: [random-nodes](specs/2026-06-10-random-nodes.md)(过 2 轮 AI 审)。

**之后(或并行设计): 给阶段2/3/4 写 plan** — 三份 spec 都过审收敛, 但还没 plan。
- **阶段2 数组**[spec](specs/2026-06-10-collection-nodes.md)(过 2 轮): 动框架(List 类型 + ForEach)。⚠ **写 plan/动代码前必先做 spec 里的 A' 框架审计 gate**(List 过 SetVar/快照/Expr/RPC/类型兼容矩阵, 命中阻塞回炉)。9 节点。
- **阶段3 数学**[spec](specs/2026-06-10-math-nodes.md)(过 1 轮): 进 purefunc 无框架改动, 9 节点 + Expr 补 6 函数。**最简单, plan 可照 random plan 套路快出**。
- **阶段4 字符串**[spec](specs/2026-06-10-string-nodes.md)(过 3 轮): 进 purefunc, 10 节点 + **改现有 Length 为 rune-based**(风险已核极低)。位置类 rune、正则错误走编辑期校验+Log.Warn。

**全局执行顺序**: 阶段1 → (2/3/4 独立, 但 2 的 RandomChoice 等 1)。阶段3/4 无框架风险最好落。
已知预存失败(非回归): runtime 缺 fish fixture, 见 [build.md](checklists/build.md)。

旧候选(无紧迫, 路线图之外): 搜索/大复合 modal 是否收进 BaseModal; idea 池(cv-perception · editor-footgun · misc-tools)。

## Hanging tasks

- [ ] 无阻塞待办。（子图 "(子图未找到)/__missing__" 反复 bug 已根治+真机验，全程入 [keepalive incident](incidents/2026-06-09-keepalive-singleton-subgraph-store-stale.md) + [import-cache incident](incidents/2026-06-09-import-bypasses-container-store-cache.md)。原积压已路由：编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md)；bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md)；i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。）
