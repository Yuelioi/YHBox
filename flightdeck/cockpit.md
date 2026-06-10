# Cockpit — YHFish

**Last updated**: 2026-06-11 by 月离 (editor-ux-v2 落地: hover 裁剪修复 + 全屏 modal + signature help/补全 info/类型色点/inline 行号 + 参考抽屉; 6 批自动门全绿, 真机视觉自检待验。)
**Active focus**: **editor-ux-v2 已实现落地, 真机视觉自检待用户** — 接上版"观感升级"回访意见 (风格认可但功能弱+文档简陋+hover bug), 6 批全落 (commit 22cb6e9→6047247), 自动门全绿 (typecheck / 247 测试 / lint 16 基线 / i18n 39 基线 / build)。**下一步走 verify 真机看 5 批, 重点: F1 抽屉键是否被 webview 拦成帮助(拦则换键)、补全类型色点默认 glyph 有没被盖、签名浮层与补全弹窗共存是否吵**。细节 [archive/specs/2026-06-11-editor-ux-v2.md](archive/specs/2026-06-11-editor-ux-v2.md) + plan 同名。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

- 无专项进行中。

## 下一步

**首选: 真机自检 editor-ux-v2** —— 走 [archive/plans/2026-06-11-editor-ux-v2.md](archive/plans/2026-06-11-editor-ux-v2.md) 的 verify 5 批 (①hover 不裁 ②全屏/行号/色点 ③补全 info/Expr 参数模板 Tab 跳 ④参考抽屉 + **F1 键** ⑤signature help)。过了清 verify 账; 若 F1 被拦 / 色点露默认 glyph / 签名与补全打架 → 小修跟进。

**"内容太少"原诉求已转译落地**: 用户最初模糊说"内容太少", 澄清后 = 样式(参考栏粗糙)+ 功能弱(缺 signature help 等), **非** i18n 文档覆盖率 —— 本批做的就是这些。若真机后用户仍觉少, 再问是否指节点/函数 description 覆盖率。

**之后候选**(无紧迫): 搜索/大复合 modal 是否收进 BaseModal; 脚本调子图 (Script 非目标遗留); idea 池(cv-perception · editor-footgun · misc-tools); oxlint 预存 16 错清债; residue 28 处 HUD/Launcher 硬编码中文(misc-tools-backlog 在册)。

**表达式系列全部收口**(已真机确认): 变量引用最终形态 = **`$hp` 语法**; EditorModal 统一壳承载 Expr/Script 放大编辑 (本批起默认全屏)。

## 待复核

- ⚠待复核: [docs/node-system-architecture.md](docs/node-system-architecture.md) — RegionRunner/Evaluator 例子清单过期(列了不存在的 Try/GetSys、漏 ForEach、PureFunc 数旧); 未记框架增量: per-dispatch evalCache(`IsNonDeterministic` 单一 gate)、`Spec.DynamicInputs` 标志、Script 节点、`List` pin 类型 + `in.List` + `node.LooseEqual/FormatValue`、validator `INVALID_REGEX_PATTERN`、`Length` rune 计数(这批原记在 cockpit 关键上下文, 2026-06-11 清版后唯一去处是这里)。when_to_update 多次命中。
- ⚠待复核: [docs/variable-system.md](docs/variable-system.md) — 正文是空壳(只有 frontmatter + 标题, 入库时就这样)。要么补正文要么删掉, 别让路由指到空文档; 补的话把 list 类型 + 类型消费点审计表(见 archive/specs/2026-06-10-list-var-type.md)一并写进去。
- ⚠待复核: [docs/expression-system.md](docs/expression-system.md) + [docs/script-system.md](docs/script-system.md) — editor-ux-v2 给 ExprInput/CodeInput 新增了 signature help / 补全 info 面板 / 类型色点 / inline 行号 / 参考抽屉(Expr 参考栏还加了参数模板), 两文档的"编辑器/补全"章节未覆盖这些。补时参 archive/specs/2026-06-11-editor-ux-v2.md。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。本节只放**两边都装不下的活上下文**, 现在没有。(2026-06-11 清版: 原"加节点路线图存档指针"等五条已分别沉淀进 docs / 待复核 / build.md, 不再重复。)
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **39** = 28(misc-tools-backlog) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 16 错(oxlint 1.64 新规则, 散在 10 个未涉编辑器的文件)。跑全套测试/检查时按此判红。
