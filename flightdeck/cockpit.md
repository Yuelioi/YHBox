# Cockpit — YHFish

**Last updated**: 2026-06-10 by 月离 (list 变量类型落地归档(spec+plan), 全套验证绿; 待用户 2 笔真机 smoke: list 变量 + 原定列表批。)
**Active focus**: 等**用户真机 smoke 2 笔**(list 变量 + collection 批, 见「待验证」)。候选拍板: Expr 语法提示要不要立项。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

- 无专项进行中。

## 下一步

**首选: 用户真机 smoke 两笔**(一次 app 启动可全验):
1. **list 变量**: 变量面板新建变量选 list 类型, 默认值填 `[1,2,3]` → 拖到画布出 GetVar → 连 ListContains(Value 填 1) → Log → 出 **true**; 再把默认值改成 `[1,2`(残缺) → 输入框变红、悬停有提示、值不保存。
2. **原定列表批**(上轮没确认): Split(a,b,c) → ForEach(元素存 item, 变量类型 any) → 读变量 → Log → 依次出 a/b/c; 未连线 List pin 显示「由连线提供」。
验完报我, 我清 verify 标记。

**候选拍板**: Expr 语法提示要不要立项(用户反馈裸文本难用, 想要函数补全/提示)。
**之后候选**(无紧迫): 搜索/大复合 modal 是否收进 BaseModal; idea 池(cv-perception · editor-footgun · misc-tools); residue 28 处 HUD/Launcher 硬编码中文(misc-tools-backlog 在册)。

## 待复核

- ⚠待复核: [docs/node-system-architecture.md](docs/node-system-architecture.md) — RegionRunner/Evaluator 例子清单过期(列了不存在的 Try/GetSys、漏 ForEach、PureFunc 数旧), dispatch 流程未记 per-dispatch evalCache。when_to_update 命中(本次改了 dispatch/RegionRunner)。
- ⚠待复核: [docs/variable-system.md](docs/variable-system.md) — 正文是空壳(只有 frontmatter + 标题, 入库时就这样)。要么补正文要么删掉, 别让路由指到空文档; 补的话把 list 类型 + 类型消费点审计表(见 archive/specs/2026-06-10-list-var-type.md)一并写进去。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文(加节点路线图存档指针)

- 全程: 4 spec + 4 plan 在 `archive/specs|plans/2026-06-10-*-nodes.md`(含各 spec 落地修订与 A' 审计结论)。31 节点: 随机 4(含 RandomChoice) + 数学 9 + 字符串 10 + 列表 8(ForEach+7)。
- 后续增量: **list 变量类型**(第 6 种, 值 `[]any`, JSON 数组默认值编辑器) 在 `archive/specs|plans/2026-06-10-list-var-type.md` — 含 VarDecl.Type 全消费点审计表; 顺路修了缺失的 `var.any_independent_placeholder` i18n key。
- 框架增量: `IsNonDeterministic` + `evalPureDataCached` per-dispatch 缓存(单一 gate, 评审 C1 教训入 [consumer-audit-gap incident](incidents/2026-05-29-storage-convention-consumer-audit-gap.md) Case 2); `List` pin 类型 + `in.List` + `node.LooseEqual/FormatValue`(不可比防护); Expr +6 函数; validator `INVALID_REGEX_PATTERN`; 现有 `Length` 改 rune 计数。
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue 28。
