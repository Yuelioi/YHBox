# Cockpit — YHFish

**Last updated**: 2026-06-10 by 月离 (Expr 随机函数落地归档: rand()/randint(min,max) + Expr 挂 IsNonDeterministic; Expr 语法提示 smoke 已过 verify 已清; 待用户 1 笔真机 smoke。)
**Active focus**: 等**用户真机 smoke 1 笔**(Expr 随机函数, 见「待验证」)。无新专项。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

- 无专项进行中。

## 下一步

**首选: 用户真机 smoke Expr 随机函数**(30 秒): Expr 表达式框敲 `ra` → 补全列表见 `rand()` / `randint(min, max)`; 写 `randint(1, 6)` 接 Log 跑两次 → 两次都落 1~6 (大概率不同值)。验完报我清 verify。

**之后候选**(无紧迫): 搜索/大复合 modal 是否收进 BaseModal; idea 池(cv-perception · editor-footgun · misc-tools); residue 28 处 HUD/Launcher 硬编码中文(misc-tools-backlog 在册)。

## 待复核

- ⚠待复核: [docs/node-system-architecture.md](docs/node-system-architecture.md) — RegionRunner/Evaluator 例子清单过期(列了不存在的 Try/GetSys、漏 ForEach、PureFunc 数旧), dispatch 流程未记 per-dispatch evalCache。when_to_update 命中(本次改了 dispatch/RegionRunner)。
- ⚠待复核: [docs/variable-system.md](docs/variable-system.md) — 正文是空壳(只有 frontmatter + 标题, 入库时就这样)。要么补正文要么删掉, 别让路由指到空文档; 补的话把 list 类型 + 类型消费点审计表(见 archive/specs/2026-06-10-list-var-type.md)一并写进去。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文(加节点路线图存档指针)

- 全程: 4 spec + 4 plan 在 `archive/specs|plans/2026-06-10-*-nodes.md`(含各 spec 落地修订与 A' 审计结论)。31 节点: 随机 4(含 RandomChoice) + 数学 9 + 字符串 10 + 列表 8(ForEach+7)。
- 后续增量: **list 变量类型**(第 6 种, 值 `[]any`, JSON 数组默认值编辑器) 在 `archive/specs|plans/2026-06-10-list-var-type.md` — 含 VarDecl.Type 全消费点审计表; 顺路修了缺失的 `var.any_independent_placeholder` i18n key。**Expr 语法提示** 在 `archive/specs|plans/2026-06-10-expr-editor-hints.md` — expr 包 builtin 函数表成单一来源(`Builtins()`/`CallRefs`), validator 新增 EXPR_UNKNOWN_FUNCTION/EXPR_FN_ARITY, widget kind `expr` → ExprInput 组件(死代码 ExpressionInput 已删)。**Expr 随机函数** 在 `archive/specs|plans/2026-06-10-expr-random-fns.md` — rand()/randint() 进函数表 12 项, Expr 挂 `IsNonDeterministic`(per-dispatch 记忆化保同帧多路径同值, 顺带修了 now() 同帧不稳)。
- 框架增量: `IsNonDeterministic` + `evalPureDataCached` per-dispatch 缓存(单一 gate, 评审 C1 教训入 [consumer-audit-gap incident](incidents/2026-05-29-storage-convention-consumer-audit-gap.md) Case 2); `List` pin 类型 + `in.List` + `node.LooseEqual/FormatValue`(不可比防护); Expr +6 函数; validator `INVALID_REGEX_PATTERN`; 现有 `Length` 改 rune 计数。
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue 28。
