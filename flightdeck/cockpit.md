# Cockpit — YHFish

**Last updated**: 2026-06-10 by 月离 (子图 `__missing__` 反复 bug **根治(第4次复发, 真机验过)**: ① 防火墙 onConnect 拦哨兵 pin、永不进存盘边 ② onActivated 切回容器重拉子图。教训进 [incident](incidents/2026-06-09-keepalive-singleton-subgraph-store-stale.md)。｜前: modal+HUD 风格统一(小 HUD 彩色面板 / modal BaseModal 纯黑平铺)。)
**Active focus**: **加节点路线图**(用户 2026-06-10 起)— 4 阶段全部已出 spec, **均未执行**(用户 token 紧, 只设计不跑)。顺序: ①随机 ②数组(提前) ③数学 ④字符串。阶段1 另有完整 plan。近期完成: 子图 `__missing__` 根治(真机验)、资产子系统(→ [docs/asset-subsystem.md](docs/asset-subsystem.md))、modal+HUD 风格统一。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

- 无专项进行中。(modal+HUD 风格统一已完成, 见 Active focus + [ui.md](checklists/ui.md) / [standalone-window-style.md](checklists/standalone-window-style.md)。)

## 下一步

**加节点路线图 — 4 spec 就绪, 待执行**(用户 token 紧时再跑):
- 阶段1 随机: [spec](specs/2026-06-10-random-nodes.md) + [plan](plans/2026-06-10-random-nodes.md)(8 任务 TDD)。含框架 per-dispatch 求值稳定(IsNonDeterministic)。已过 2 轮外部 AI 审。
- 阶段2 数组: [spec](specs/2026-06-10-collection-nodes.md)。动框架(List 类型+ForEach)。9 节点。RandomChoice 归 random 包。
- 阶段3 数学: [spec](specs/2026-06-10-math-nodes.md)。进 purefunc, 无框架改动。9 节点 + Expr 补 6 函数。
- 阶段4 字符串: [spec](specs/2026-06-10-string-nodes.md)。进 purefunc。10 节点(位置类 rune-based)。
- **执行顺序**: 阶段1 先(阶段2 的 RandomChoice 依赖它)。阶段2 写 plan 前可再丢 AI 审一轮。已知预存失败(非回归): runtime 缺 fish fixture, 见 [build.md](checklists/build.md)。

旧候选(无紧迫): 搜索/大复合 modal 是否收进 BaseModal; idea 池(cv-perception · editor-footgun · misc-tools)。

## Hanging tasks

- [ ] 无阻塞待办。（子图 "(子图未找到)/__missing__" 反复 bug 已根治+真机验，全程入 [keepalive incident](incidents/2026-06-09-keepalive-singleton-subgraph-store-stale.md) + [import-cache incident](incidents/2026-06-09-import-bypasses-container-store-cache.md)。原积压已路由：编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md)；bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md)；i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。）
