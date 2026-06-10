# Cockpit — YHFish

**Last updated**: 2026-06-11 by 月离 (Script 节点全链路落地+graduate 进 docs/script-system; 用户反馈后补齐: Expr/Script 统一放大编辑 modal(参考面板/点击插入/实时 lint); 挂 2 笔真机 smoke。)
**Active focus**: **Script 节点 + 放大编辑 modal 已落地待真机 smoke**(见「下一步」, 连同 CodeMirror smoke 一起验); 验完议「变量绑定路线 A」(先确认 Script vars.get 缓解后还做不做)。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

- 无专项进行中。

## 下一步

**首选: 真机 smoke ×2, 一次跑完报我清 verify**:
1. **Script 节点 + 放大编辑 modal**(9 步): ① 面板/右键/explorer 搜「脚本」; ② 写 `log.info("hi"); return 1 + 1`, CaptureResult 填变量, 跑容器看日志 hi、GetVar 读到 2; ③ `while(true){}` 跑起来点停止立即停; ④ `ClickAt({XRatio: 0.5, YRatio: 0.5})` 真点击; ⑤ Inspector 点放大 → 大 modal: 敲 `Wait` 出补全, **右侧面板搜「点击」点一下插入 `ClickAt({})`**, Ctrl+Enter 确认回写; ⑥ 故意写 `let a = ;` 节点红错带行号; ⑦ Expr/Script 的 Inspector「输入口」区加 `hp:number` 出引脚可连线; ⑧ **Expr 小框也有放大按钮**, modal 右侧列全部函数(签名+说明)点击插入; ⑨ Expr modal 里写 `clmap(1` 底部状态栏实时红字。
2. **CodeMirror 表达式编辑器**(4 步, 一并覆盖随机函数): ① Expr 写 `"abc" + 1` 看高亮; ② 敲 `ra` 补全 Tab 上屏; ③ `clmap(1)` 红波浪线+悬停; ④ `randint(1, 6)` 接 Log 跑两次落 1~6。

**之后议(表达式遗留议题)**:
1. **变量绑定进 Expr (路线 A, 已细化)**: Inputs[] 项加可选 `Var`+`Scope` 字段 = 绑定变量: ① Expr.Evaluate 构 env 绑定项直读 `ctx.Vars().GetScoped()` (零框架改动); ② FE dataInDynamicFn 跳过绑定项 (不渲染引脚), 声明编辑加来源下拉+VarNameInput (声明编辑区本身已由 DynamicInputsEditor 落地, 路线 A 在它上面扩展); ③ validator 绑定项跳 DATA_PIN_DANGLING、加变量存在性+类型检查; ④ 节点卡片列绑定变量名小字。**B** (恢复 `$vars.*` 语法) 走前必须先翻 v4 删语法的完整理由 (头号铁律); 自动建 GetVar 方案已淘汰。注: Script 的 `vars.get` 已缓解部分痛点, 议时先确认路线 A 还要不要做。(原议题「Expr 放大编辑 modal」已于 2026-06-11 随 EditorModal 统一壳落地, 不再议。)

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
