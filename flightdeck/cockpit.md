# Cockpit — YHFish

**Last updated**: 2026-06-11 by 月离 (文档复核: 4 份待复核文档债全部销账 — node-system-architecture 修例子错+补框架增量、variable-system 空壳补全、expression+script-system 补 signature help/类型色点/修 scrollPastEnd, 均对源码核过 stale→active。前序当日: 看板清理 + 三项 verify 销账。下一程不变: 编辑器参数提示补 enum 值建议。)
**Active focus**: **下个会话: 编辑器参数提示补 enum 值建议** — 节点函数的枚举参数 (用户例: `GetVar({VarName, Scope})` 里 Scope 只能 local/auto/global) 在脚本/表达式编辑器里打值时应**补全/提示可选值**, 现状只给签名不给候选。数据源 = 节点 Spec 的 dropdown widget options (开工先 grep 确认 Scope pin 的 options 怎么存)。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

- 无专项进行中。

**首选 (下个会话): 编辑器参数提示补 enum 值建议** —— 节点函数枚举参数 (例 `GetVar({Scope})` → local/auto/global) 在 Script/Expr 编辑器里打值位置应补全候选值 / 签名里列出可选集, 现在只给签名不给值。第一步: grep 确认这类 enum pin (Scope 等) 的可选值在 Spec 里怎么存 (dropdown widgetKind + options? 还是 backend 常量?), 设计补全/签名怎么读到。问清范围再开 spec。

**之后候选**(无紧迫): 搜索/大复合 modal 是否收进 BaseModal; 脚本调子图 (Script 非目标遗留); idea 池(cv-perception · editor-footgun · misc-tools); oxlint 预存 16 错清债; residue 28 处 HUD/Launcher 硬编码中文(misc-tools-backlog 在册)。

## 待复核

- 无。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。本节只放**两边都装不下的活上下文**, 现在没有。(2026-06-11 清版: 原"加节点路线图存档指针"等五条已分别沉淀进 docs / 待复核 / build.md, 不再重复。)
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **39** = 28(misc-tools-backlog) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 16 错(oxlint 1.64 新规则, 散在 10 个未涉编辑器的文件)。跑全套测试/检查时按此判红。
