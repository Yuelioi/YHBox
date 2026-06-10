# Cockpit — YHFish

**Last updated**: 2026-06-10 by 月离 (表达式三连落地归档(RPC 单源/docs/CodeMirror); 用户收尾抛两问(表达式 modal 编辑 + 变量引用)留待下个对话先议。)
**Active focus**: 下个对话先议**表达式两问**(modal 大编辑器 / 变量引用进表达式, 见「下一步」), 议完立项; 另挂 1 笔 CodeMirror 真机 smoke。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

- 无专项进行中。

## 下一步

**首选: 用户 2026-06-10 收尾提出的两个表达式问题, 下个对话先议**(用户原话记录):
1. **"表达式都在一个 textarea 里操作不方便, 为什么不是一个单独的 modal"** — 候选方案: Inspector 小框旁加「放大编辑」按钮弹大 modal, 里面复用现成 CodeMirror ExprInput (组件 API 已解耦, 改动小); 顺路跟旧候选「搜索/大复合 modal 收进 BaseModal」一起考虑。
2. **"不支持变量系统么"** — 现状是**设计决策不是缺陷**: v4 删了 `$vars.*` 语法, 变量必须 GetVar 节点连进 Expr 动态输入 (见 [docs/expression-system.md](docs/expression-system.md))。用户觉得绕。候选方向: (a) 补全里列容器变量, 选中自动声明 Inputs + 建 GetVar 连线 (编辑器侧自动化, 不动语法); (b) 恢复某种变量引用语法 (动 runtime, 影响面大)。议完再立项。

**还挂着: 真机 smoke CodeMirror 表达式编辑器**(1 分钟, 一并覆盖随机函数): ① Expr 写 `"abc" + 1` 看高亮; ② 敲 `ra` 补全 Tab 上屏; ③ `clmap(1)` 红波浪线+悬停提示; ④ `randint(1, 6)` 接 Log 跑两次落 1~6。验完报我清 verify。

**之后候选**(无紧迫): 搜索/大复合 modal 是否收进 BaseModal; idea 池(cv-perception · editor-footgun · misc-tools); residue 28 处 HUD/Launcher 硬编码中文(misc-tools-backlog 在册)。

## 待复核

- ⚠待复核: [docs/node-system-architecture.md](docs/node-system-architecture.md) — RegionRunner/Evaluator 例子清单过期(列了不存在的 Try/GetSys、漏 ForEach、PureFunc 数旧), dispatch 流程未记 per-dispatch evalCache。when_to_update 命中(本次改了 dispatch/RegionRunner)。
- ⚠待复核: [docs/variable-system.md](docs/variable-system.md) — 正文是空壳(只有 frontmatter + 标题, 入库时就这样)。要么补正文要么删掉, 别让路由指到空文档; 补的话把 list 类型 + 类型消费点审计表(见 archive/specs/2026-06-10-list-var-type.md)一并写进去。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文(加节点路线图存档指针)

- 全程: 4 spec + 4 plan 在 `archive/specs|plans/2026-06-10-*-nodes.md`(含各 spec 落地修订与 A' 审计结论)。31 节点: 随机 4(含 RandomChoice) + 数学 9 + 字符串 10 + 列表 8(ForEach+7)。
- 后续增量: **list 变量类型**(第 6 种, 值 `[]any`, JSON 数组默认值编辑器) 在 `archive/specs|plans/2026-06-10-list-var-type.md` — 含 VarDecl.Type 全消费点审计表; 顺路修了缺失的 `var.any_independent_placeholder` i18n key。**Expr 语法提示** 在 `archive/specs|plans/2026-06-10-expr-editor-hints.md` — expr 包 builtin 函数表成单一来源(`Builtins()`/`CallRefs`), validator 新增 EXPR_UNKNOWN_FUNCTION/EXPR_FN_ARITY, widget kind `expr` → ExprInput 组件(死代码 ExpressionInput 已删)。**Expr 随机函数** 在 `archive/specs|plans/2026-06-10-expr-random-fns.md` — rand()/randint() 进函数表 12 项, Expr 挂 `IsNonDeterministic`(per-dispatch 记忆化保同帧多路径同值, 顺带修了 now() 同帧不稳)。**表达式三连**(2026-06-10 晚): 函数元数据 RPC 单源(`expr.Functions()` → GetExprFunctions, FE 手写表已删, 加函数=Go 表一处+i18n 两条) + 常驻文档 [docs/expression-system.md](docs/expression-system.md) + ExprInput 换 CodeMirror 6(高亮/带位置红线/原生补全; 新依赖 @codemirror/*), 在 `archive/specs|plans/2026-06-10-expr-{fn-rpc-single-source,codemirror-editor}.md`。
- 框架增量: `IsNonDeterministic` + `evalPureDataCached` per-dispatch 缓存(单一 gate, 评审 C1 教训入 [consumer-audit-gap incident](incidents/2026-05-29-storage-convention-consumer-audit-gap.md) Case 2); `List` pin 类型 + `in.List` + `node.LooseEqual/FormatValue`(不可比防护); Expr +6 函数; validator `INVALID_REGEX_PATTERN`; 现有 `Length` 改 rune 计数。
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue 28。
