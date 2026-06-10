# Cockpit — YHFish

**Last updated**: 2026-06-10 by 月离 (**加节点路线图 阶段1/3/4 全部实现落地**(subagent-driven TDD + 双阶段审 + 终审), 阶段2 数组进行中: A' 框架审计 gate 跑批。｜前: 4 spec 收敛零代码。)
**Active focus**: **加节点路线图**(用户 2026-06-10 起)— **阶段1 随机 ✅ / 阶段3 数学 ✅ / 阶段4 字符串 ✅ 已实现并过终审**(细节见「进行中」), 阶段2 数组在 A' 审计 gate。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

- **阶段2 数组**: A' 框架审计 gate 进行中(3 路并行: 后端链路 / 前端类型矩阵 / 持久化编辑链)。spec: [collection-nodes](specs/2026-06-10-collection-nodes.md)。审计无阻塞 → 写 plan → subagent 执行; 命中阻塞 → spec 回炉。

## 下一步

### 🎯 加节点路线图进度 (2026-06-10 本对话)

**已落地(全部 subagent-driven TDD + 每任务 spec 审 + 阶段终审):**
- **阶段1 随机 ✅**: RandomInt/Float/Bool + `IsNonDeterministic` + per-dispatch evalCache。9 commits (2f093cb..13f6ae7)。**终审抓到并修复 Critical C1**: 缓存最初只挂 evalDataSource, 被主路径 resolveDataPinV5 绕过 — 已并成 `evalPureDataCached` 单一 gate (50de637)。spec 已补"调用点缺口"修订。RandomInt Min/Max 落地改 Number (撞 DetectNameSplits 守卫, spec 已记修订)。
- **阶段3 数学 ✅**: purefunc +9 节点 (Abs/Min/Max/Floor/Ceil/Round/Clamp/Pow/Sqrt) + Expr 补 6 函数 + FN_NAMES/Expr-description 同步 (spec 漏的两站点, plan 补上)。plan: [math-nodes](plans/2026-06-10-math-nodes.md)。6 commits (fea0d87..6402c4b)。
- **阶段4 字符串 ✅**: purefunc +10 节点 + **Length 改 rune 计数** + validator `INVALID_REGEX_PATTERN` 编辑期校验。plan: [string-nodes](plans/2026-06-10-string-nodes.md)。7 commits (3ceb6ae..f1d4a5d)。终审修了 Length 旧字节文案。
- **真机 smoke 全部留给用户**(各 plan 末任务有"看什么/怎么验/什么算过")。

**进行中: 阶段2 数组** — 见「进行中」。
已知预存失败(非回归): runtime 缺 fish fixture, 见 [build.md](checklists/build.md); i18n residue 28 处 (HUD/Launcher 硬编码)。

旧候选(无紧迫, 路线图之外): 搜索/大复合 modal 是否收进 BaseModal; idea 池(cv-perception · editor-footgun · misc-tools)。

## Hanging tasks

- [ ] 无阻塞待办。（子图 "(子图未找到)/__missing__" 反复 bug 已根治+真机验，全程入 [keepalive incident](incidents/2026-06-09-keepalive-singleton-subgraph-store-stale.md) + [import-cache incident](incidents/2026-06-09-import-bypasses-container-store-cache.md)。原积压已路由：编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md)；bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md)；i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。）
