# Cockpit — YHFish

**Last updated**: 2026-06-11 by 月离 (变量绑定路线 A 落地归档 — 表达式两问议题全部关闭; 编辑器三轮反馈打磨完; 挂 3 笔真机 smoke。)
**Active focus**: **Script 节点 + 编辑器 + 变量绑定 已全部落地, 待真机 smoke ×3**(见「下一步」)。验完无挂账, 回 idea 池挑下一个方向。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

- 无专项进行中。

## 下一步

**首选: 真机 smoke ×3, 一次跑完报我清 verify**:
0. **变量绑定输入**(3 步): ① 容器声明变量 hp=42, Expr「输入口」加行 a (来源默认「绑定变量」) 选 hp → Type 自动 number、卡片出小字 `a ← hp`、无 a 引脚, 表达式 `a + 1` 跑出 43; ② 绑不存在的变量 → 红错; ③ 来源切「连线」→ 引脚出现。
1. **Script 节点 + 放大编辑 modal**(12 步): ① 面板/右键/explorer 搜「脚本」; ② 写 `log.info("hi"); return 1 + 1`, CaptureResult 填变量, 跑容器看日志 hi、GetVar 读到 2; ③ `while(true){}` 跑起来点停止立即停; ④ `ClickAt({XRatio: 0.5, YRatio: 0.5})` 真点击; ⑤ Inspector 点放大 → 大 modal: 敲 `Wait` 出补全(VSCode 风格提示框), **右侧面板节点按分类分组带配色**, 行点一下展开(说明+参数表+示例), 行尾按钮插入 → **落成 `ClickAt({XRatio: …, YRatio: …}) 占位, Tab 逐格填值**, Ctrl+Enter 确认回写; ⑥ 故意写 `let a = ;` 节点红错带行号; ⑦ Expr/Script 的 Inspector「输入口」区加 `hp:number` 出引脚可连线; ⑧ **Expr 小框也有放大按钮**, modal 右侧列全部函数(签名+说明)点击插入; ⑨ Expr modal 里写 `clmap(1` 底部状态栏实时红字; ⑩ **`vars.get("` 里出变量名补全**; ⑪ 面板「变量」组置顶点击插 vars.get, **工具栏右侧「新建变量」**(建完自动插一句); ⑫ 工具栏: 片段下拉插 for 循环、`//` 注释当前行、**查找替换面板是暗色中文样式**; 底部状态栏显示行数·字符数。
2. **CodeMirror 表达式编辑器**(4 步, 一并覆盖随机函数): ① Expr 写 `"abc" + 1` 看高亮; ② 敲 `ra` 补全 Tab 上屏; ③ `clmap(1)` 红波浪线+悬停; ④ `randint(1, 6)` 接 Log 跑两次落 1~6。

**表达式两问全部落地** (原议题关闭): 「Expr 放大编辑 modal」随 EditorModal 统一壳落地; 「变量绑定路线 A」2026-06-11 用户拍板后落地 (绑定为默认, 连线保留兜底 — **"砍连线模式"** 被搁置: Expr→Expr 链与 fusion 依赖它, 用户日后想砍单独议)。

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
