# Cockpit — YHFish

**Last updated**: 2026-06-11 by 月离 (编辑器观感升级落地: 用户拍板 C 案 VSCode Dark+ + 功能全做, 实施+验证+离屏视觉自检完成, spec/plan 归档, 待真机过目。)
**Active focus**: **编辑器观感升级已交付, 待用户真机过目** (verify 项见归档 spec)。三处编辑器 (ExprInput/CodeInput/EditorModal) 共享 `lib/editorTheme.ts` 成套主题 (VSCode Dark+; $变量例外橙标) + 基础手感件; 智能层 = 语法快速反馈/未声明 $变量/hover 文档/折叠/Ctrl+D; modal 7xl+全屏档; 字体 JetBrains Mono+Inter 本地打包。细节进 [docs/script-system.md](docs/script-system.md) / [docs/expression-system.md](docs/expression-system.md)。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

- 无专项进行中。

## 下一步

**首选: 用户真机过目编辑器升级** (verify 项随归档 spec 在册, 见待验证扫描)。看完没问题即清账; 有不满意处直接报, 改动集中在 `lib/editorTheme.ts` 一张配色/chrome 表。

**之后候选**(无紧迫): 搜索/大复合 modal 是否收进 BaseModal; 脚本调子图 (Script 非目标遗留); idea 池(cv-perception · editor-footgun · misc-tools); oxlint 预存 16 错清债(见关键上下文)。

**表达式系列全部收口**(已真机确认): 变量引用最终形态 = **`$hp` 语法** (用户拍板推翻 v4, 决策史与依据在 [docs/expression-system.md](docs/expression-system.md) — 绑定模式被它取代删除, 输入口退化为纯连线引脚); EditorModal 统一壳承载 Expr/Script 放大编辑。

**之后候选**(无紧迫): 搜索/大复合 modal 是否收进 BaseModal; 脚本调子图 (Script 非目标遗留); idea 池(cv-perception · editor-footgun · misc-tools); residue 28 处 HUD/Launcher 硬编码中文(misc-tools-backlog 在册)。

## 待复核

- ⚠待复核: [docs/node-system-architecture.md](docs/node-system-architecture.md) — RegionRunner/Evaluator 例子清单过期(列了不存在的 Try/GetSys、漏 ForEach、PureFunc 数旧); 未记框架增量: per-dispatch evalCache(`IsNonDeterministic` 单一 gate)、`Spec.DynamicInputs` 标志、Script 节点、`List` pin 类型 + `in.List` + `node.LooseEqual/FormatValue`、validator `INVALID_REGEX_PATTERN`、`Length` rune 计数(这批原记在 cockpit 关键上下文, 2026-06-11 清版后唯一去处是这里)。when_to_update 多次命中。
- ⚠待复核: [docs/variable-system.md](docs/variable-system.md) — 正文是空壳(只有 frontmatter + 标题, 入库时就这样)。要么补正文要么删掉, 别让路由指到空文档; 补的话把 list 类型 + 类型消费点审计表(见 archive/specs/2026-06-10-list-var-type.md)一并写进去。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。本节只放**两边都装不下的活上下文**, 现在没有。(2026-06-11 清版: 原"加节点路线图存档指针"等五条已分别沉淀进 docs / 待复核 / build.md, 不再重复。)
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **39** = 28(misc-tools-backlog) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 16 错(oxlint 1.64 新规则, 散在 10 个未涉编辑器的文件)。跑全套测试/检查时按此判红。
