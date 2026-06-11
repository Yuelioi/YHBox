# Cockpit — YHFish

**Last updated**: 2026-06-11 by 月离 (看板清理: 配色统一 / 用户片段 prefix / 编辑器视觉底座三项 verify 销账 — 用户真机确认; cockpit 收掉已落地里程碑块。下一程不变: 编辑器参数提示补 enum 值建议。)
**Active focus**: **下个会话: 编辑器参数提示补 enum 值建议** — 节点函数的枚举参数 (用户例: `GetVar({VarName, Scope})` 里 Scope 只能 local/auto/global) 在脚本/表达式编辑器里打值时应**补全/提示可选值**, 现状只给签名不给候选。数据源 = 节点 Spec 的 dropdown widget options (开工先 grep 确认 Scope pin 的 options 怎么存)。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

- 无专项进行中。

**首选 (下个会话): 编辑器参数提示补 enum 值建议** —— 节点函数枚举参数 (例 `GetVar({Scope})` → local/auto/global) 在 Script/Expr 编辑器里打值位置应补全候选值 / 签名里列出可选集, 现在只给签名不给值。第一步: grep 确认这类 enum pin (Scope 等) 的可选值在 Spec 里怎么存 (dropdown widgetKind + options? 还是 backend 常量?), 设计补全/签名怎么读到。问清范围再开 spec。

**之后候选**(无紧迫): 搜索/大复合 modal 是否收进 BaseModal; 脚本调子图 (Script 非目标遗留); idea 池(cv-perception · editor-footgun · misc-tools); oxlint 预存 16 错清债; residue 28 处 HUD/Launcher 硬编码中文(misc-tools-backlog 在册)。

## 待复核

- ⚠待复核: [docs/node-system-architecture.md](docs/node-system-architecture.md) — RegionRunner/Evaluator 例子清单过期(列了不存在的 Try/GetSys、漏 ForEach、PureFunc 数旧); 未记框架增量: per-dispatch evalCache(`IsNonDeterministic` 单一 gate)、`Spec.DynamicInputs` 标志、Script 节点、`List` pin 类型 + `in.List` + `node.LooseEqual/FormatValue`、validator `INVALID_REGEX_PATTERN`、`Length` rune 计数(这批原记在 cockpit 关键上下文, 2026-06-11 清版后唯一去处是这里)。when_to_update 多次命中。
- ⚠待复核: [docs/variable-system.md](docs/variable-system.md) — 正文是空壳(只有 frontmatter + 标题, 入库时就这样)。要么补正文要么删掉, 别让路由指到空文档; 补的话把 list 类型 + 类型消费点审计表(见 archive/specs/2026-06-10-list-var-type.md)一并写进去。
- ⚠待复核: [docs/expression-system.md](docs/expression-system.md) + [docs/script-system.md](docs/script-system.md) — editor-ux-v2 给 ExprInput/CodeInput 新增了 signature help / 补全 info 面板 / 类型色点 / inline 行号 / 参考抽屉(Expr 参考栏还加了参数模板), 两文档的"编辑器/补全"章节未覆盖这些。补时参 archive/specs/2026-06-11-editor-ux-v2.md。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。本节只放**两边都装不下的活上下文**, 现在没有。(2026-06-11 清版: 原"加节点路线图存档指针"等五条已分别沉淀进 docs / 待复核 / build.md, 不再重复。)
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **39** = 28(misc-tools-backlog) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 16 错(oxlint 1.64 新规则, 散在 10 个未涉编辑器的文件)。跑全套测试/检查时按此判红。
