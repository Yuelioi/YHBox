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

- ⚠待复核: [docs/node-system-architecture.md](docs/node-system-architecture.md) — RegionRunner/Evaluator 例子清单过期(列了不存在的 Try/GetSys、漏 ForEach、PureFunc 数旧); 未记框架增量: per-dispatch evalCache(`IsNonDeterministic` 单一 gate)、`Spec.DynamicInputs` 标志、Script 节点、`List` pin 类型 + `in.List` + `node.LooseEqual/FormatValue`、validator `INVALID_REGEX_PATTERN`、`Length` rune 计数(这批原记在 cockpit 关键上下文, 2026-06-11 清版后唯一去处是这里)。when_to_update 多次命中。
- ⚠待复核: [docs/variable-system.md](docs/variable-system.md) — 正文是空壳(只有 frontmatter + 标题, 入库时就这样)。要么补正文要么删掉, 别让路由指到空文档; 补的话把 list 类型 + 类型消费点审计表(见 archive/specs/2026-06-10-list-var-type.md)一并写进去。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。本节只放**两边都装不下的活上下文**, 现在没有。(2026-06-11 清版: 原"加节点路线图存档指针"等五条已分别沉淀进 docs / 待复核 / build.md, 不再重复。)
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue 28(misc-tools-backlog)。跑全套测试时按此判红。
