# Cockpit — YHFish

**Last updated**: 2026-06-10 by 月离 (真机 smoke 3/4 过(随机/数学/字符串 verify 已清); 列表 smoke 用户改测「any 变量手填 [1,2,3] → ListContains」得 false, 根因查明 = 变量系统无 list 类型, 等用户拍板。)
**Active focus**: 列表能力收尾 — 用户 smoke 发现**变量系统装不了手填列表**(根因已查明, 非 ListContains bug), 等拍板是否加 list 变量类型; 另两笔候选: Expr 语法提示、原定 Split→ForEach smoke 未确认跑过。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

- 无专项进行中。(加节点路线图 4 spec + 4 plan 全部 done 已入 archive/。)

## 下一步

**首选: 用户拍板三件事**(都来自 2026-06-10 smoke 反馈):
1. **是否加 `list` 变量类型**: 现状 VarDecl 只有 number/string/bool/point/any; any 的默认值编辑器是纯文本框, 手填 `[1,2,3]` 存的是字符串, `in.List` 不收裸串 → ListContains 永远 false(静默)。推荐加 list 类型 + JSON 数组编辑器(非法 JSON 红错)。
2. **Expr 语法提示要不要立项**: 用户反馈 Expr 编辑器裸文本难用, 想要函数补全/提示。
3. **原定列表 smoke**(Split(a,b,c)→ForEach→读变量→Log 依次出 a/b/c)跑过没? collection plan 的 verify 标记还挂着。

**之后候选**(无紧迫): 搜索/大复合 modal 是否收进 BaseModal; idea 池(cv-perception · editor-footgun · misc-tools); residue 28 处 HUD/Launcher 硬编码中文(misc-tools-backlog 在册)。

## 待复核

- ⚠待复核: [docs/node-system-architecture.md](docs/node-system-architecture.md) — RegionRunner/Evaluator 例子清单过期(列了不存在的 Try/GetSys、漏 ForEach、PureFunc 数旧), dispatch 流程未记 per-dispatch evalCache。when_to_update 命中(本次改了 dispatch/RegionRunner)。
- ⚠待复核: [docs/variable-system.md](docs/variable-system.md) — 正文是空壳(只有 frontmatter + 标题, 入库时就这样)。要么补正文要么删掉, 别让路由指到空文档。
- 小 bug 在册: `VarRow.vue:78` 引用 i18n key `var.any_independent_placeholder`, zh/en 都没这个 key — any 类型变量默认值框的 placeholder 会显示生 key。一行修, 等顺路改 VarRow 时带上。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文(加节点路线图存档指针)

- 全程: 4 spec + 4 plan 在 `archive/specs|plans/2026-06-10-*-nodes.md`(含各 spec 落地修订与 A' 审计结论)。31 节点: 随机 4(含 RandomChoice) + 数学 9 + 字符串 10 + 列表 8(ForEach+7)。
- 框架增量: `IsNonDeterministic` + `evalPureDataCached` per-dispatch 缓存(单一 gate, 评审 C1 教训入 [consumer-audit-gap incident](incidents/2026-05-29-storage-convention-consumer-audit-gap.md) Case 2); `List` pin 类型 + `in.List` + `node.LooseEqual/FormatValue`(不可比防护); Expr +6 函数; validator `INVALID_REGEX_PATTERN`; 现有 `Length` 改 rune 计数。
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue 28。
