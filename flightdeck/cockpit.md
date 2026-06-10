# Cockpit — YHFish

**Last updated**: 2026-06-10 by 月离 (加节点路线图四阶段**全部实现落地并归档**, 31 个新节点 + List/ForEach/evalCache 框架地基; 待用户 4 笔真机 smoke。)
**Active focus**: 加节点路线图已收官 — 等**用户真机 smoke**(4 笔 verify 债, 见「待复核/待验证」)。无新专项。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

- 无专项进行中。(加节点路线图 4 spec + 4 plan 全部 done 已入 archive/。)

## 下一步

**首选: 用户真机 smoke 四批新节点**(一次 app 启动可全验, 每条 30 秒):
1. 随机: 拖 RandomInt(Min=1/Max=6) 连 Log 跑一次 → 输出落 1~6; 三处面板见「随机」分组。
2. 数学: Round(X=3.14159, 位数=2) 连 Log → 出 3.14; Expr 写 `clamp(15, 0, 10)` → 出 10。
3. 字符串: Substring(中文abc, 起点0, 长度2) → 出「中文」; RegexMatch Pattern 填 `(` → 节点红错, 改 `\d+` 红错消失。
4. 列表: Split(a,b,c) → ForEach(元素存 item, 变量类型 any) → 读变量 → Log → 依次出 a/b/c; 未连线 List pin 显示「由连线提供」。
验完报我, 我清掉对应 plan 的 verify 标记。

**之后候选**(无紧迫): 搜索/大复合 modal 是否收进 BaseModal; idea 池(cv-perception · editor-footgun · misc-tools); residue 28 处 HUD/Launcher 硬编码中文(misc-tools-backlog 在册)。

## 待复核

- ⚠待复核: [docs/node-system-architecture.md](docs/node-system-architecture.md) — RegionRunner/Evaluator 例子清单过期(列了不存在的 Try/GetSys、漏 ForEach、PureFunc 数旧), dispatch 流程未记 per-dispatch evalCache。when_to_update 命中(本次改了 dispatch/RegionRunner)。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文(加节点路线图存档指针)

- 全程: 4 spec + 4 plan 在 `archive/specs|plans/2026-06-10-*-nodes.md`(含各 spec 落地修订与 A' 审计结论)。31 节点: 随机 4(含 RandomChoice) + 数学 9 + 字符串 10 + 列表 8(ForEach+7)。
- 框架增量: `IsNonDeterministic` + `evalPureDataCached` per-dispatch 缓存(单一 gate, 评审 C1 教训入 [consumer-audit-gap incident](incidents/2026-05-29-storage-convention-consumer-audit-gap.md) Case 2); `List` pin 类型 + `in.List` + `node.LooseEqual/FormatValue`(不可比防护); Expr +6 函数; validator `INVALID_REGEX_PATTERN`; 现有 `Length` 改 rune 计数。
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue 28。
