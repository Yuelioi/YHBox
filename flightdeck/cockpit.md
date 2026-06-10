# Cockpit — YHFish

**Last updated**: 2026-06-11 by 月离 (用户真机确认 Script/$ 语法/编辑器系列没问题, 4 笔 verify 清账; 用户点了下一个方向: 美化脚本编辑器。)
**Active focus**: **下个对话: 美化脚本编辑器** — 用户原话"现在的还是太简陋 太丑了" (功能已齐: 高亮/补全/参考面板/工具栏/状态栏, 痛点在**观感**)。先看现状截图议方向再动手。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

- 无专项进行中。

## 下一步

**首选: 美化脚本编辑器**(用户 2026-06-11 收尾点名, 原话"现在的还是太简陋 太丑了")。功能层已齐(JS 高亮/补全/snippet 占位/参考面板分类配色/工具栏/暗色查找/状态栏), 痛点在**视觉与质感**。入手建议: ① 先让用户截图圈出最丑的几处; ② 对照 VSCode 观感拉差距(编辑器配色主题成套化、选中行/光标行高亮、缩进参考线、字体与行距、modal 整体布局留白、面板卡片质感); ③ 改动集中在 `lib/editorTheme.ts`(共享主题)、`EditorModal.vue`、`scriptCompletions.ts` 的 HighlightStyle — 三处编辑器共用, 改一处全生效。可参考现成 CodeMirror 主题包(如 @uiw/codemirror-theme-vscode / thememirror)直接引主题替手写配色。

**表达式系列全部收口**(已真机确认): 变量引用最终形态 = **`$hp` 语法** (用户拍板推翻 v4, 决策史与依据在 [docs/expression-system.md](docs/expression-system.md) — 绑定模式被它取代删除, 输入口退化为纯连线引脚); EditorModal 统一壳承载 Expr/Script 放大编辑。

**之后候选**(无紧迫): 搜索/大复合 modal 是否收进 BaseModal; 脚本调子图 (Script 非目标遗留); idea 池(cv-perception · editor-footgun · misc-tools); residue 28 处 HUD/Launcher 硬编码中文(misc-tools-backlog 在册)。

## 待复核

- ⚠待复核: [docs/node-system-architecture.md](docs/node-system-architecture.md) — RegionRunner/Evaluator 例子清单过期(列了不存在的 Try/GetSys、漏 ForEach、PureFunc 数旧), dispatch 流程未记 per-dispatch evalCache; 2026-06-11 又加: 未记 `Spec.DynamicInputs` 标志与 Script 节点。when_to_update 再次命中。
- ⚠待复核: [docs/variable-system.md](docs/variable-system.md) — 正文是空壳(只有 frontmatter + 标题, 入库时就这样)。要么补正文要么删掉, 别让路由指到空文档; 补的话把 list 类型 + 类型消费点审计表(见 archive/specs/2026-06-10-list-var-type.md)一并写进去。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文(加节点路线图存档指针)

- 全程: 4 spec + 4 plan 在 `archive/specs|plans/2026-06-10-*-nodes.md`(含各 spec 落地修订与 A' 审计结论)。31 节点: 随机 4(含 RandomChoice) + 数学 9 + 字符串 10 + 列表 8(ForEach+7)。
- 后续增量: **list 变量类型**(第 6 种, 值 `[]any`, JSON 数组默认值编辑器) 在 `archive/specs|plans/2026-06-10-list-var-type.md` — 含 VarDecl.Type 全消费点审计表; 顺路修了缺失的 `var.any_independent_placeholder` i18n key。**Expr 语法提示** 在 `archive/specs|plans/2026-06-10-expr-editor-hints.md` — expr 包 builtin 函数表成单一来源(`Builtins()`/`CallRefs`), validator 新增 EXPR_UNKNOWN_FUNCTION/EXPR_FN_ARITY, widget kind `expr` → ExprInput 组件(死代码 ExpressionInput 已删)。**Expr 随机函数** 在 `archive/specs|plans/2026-06-10-expr-random-fns.md` — rand()/randint() 进函数表 12 项, Expr 挂 `IsNonDeterministic`(per-dispatch 记忆化保同帧多路径同值, 顺带修了 now() 同帧不稳)。**表达式三连**(2026-06-10 晚): 函数元数据 RPC 单源(`expr.Functions()` → GetExprFunctions, FE 手写表已删, 加函数=Go 表一处+i18n 两条) + 常驻文档 [docs/expression-system.md](docs/expression-system.md) + ExprInput 换 CodeMirror 6(高亮/带位置红线/原生补全; 新依赖 @codemirror/*), 在 `archive/specs|plans/2026-06-10-expr-{fn-rpc-single-source,codemirror-editor}.md`。
- **Script 节点**(2026-06-11, 7 commits de93639..b6cb89c+2eda928): 常驻知识全在 [docs/script-system.md](docs/script-system.md)(graduate 产物, 决策取舍也在); plan 含 9 任务执行记录在 `archive/plans/2026-06-10-script-node.md`(verify 挂真机 smoke)。框架侧顺带: `Spec.DynamicInputs` 标志(消 Expr kind 特判)、`ScriptBindable` 判定单一源、DynamicInputsEditor 补上 Expr 缺失的输入声明 UI、catalog drift 守卫补 event 包失明。新依赖 goja。
- 框架增量: `IsNonDeterministic` + `evalPureDataCached` per-dispatch 缓存(单一 gate, 评审 C1 教训入 [consumer-audit-gap incident](incidents/2026-05-29-storage-convention-consumer-audit-gap.md) Case 2); `List` pin 类型 + `in.List` + `node.LooseEqual/FormatValue`(不可比防护); Expr +6 函数; validator `INVALID_REGEX_PATTERN`; 现有 `Length` 改 rune 计数。
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue 28。
