# Cockpit — YHFish

**Last updated**: 2026-06-12 by 月离 (数据层大整理真机验收全过 + 收尾完成: 防呆闸/一次性迁移工具/备份全删, spec 拍 done 归档。无在飞主线。)
**Active focus**: 无在飞主线 — **数据层大整理全程收口**(spec/plan 均已归档)。下一个活从候选池挑。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

<!-- AUTO:inprogress -->
- 无
<!-- /AUTO -->

## 下一步

无紧迫主线, 候选池(用户挑): 修 2a0ff140 测试容器的预存悬空引用 sg-0d53b1bb(删那个节点即可, 顺手活); WaitTemplate 孤儿边原子性硬化(真机再现再修); 复发#5 promotion 候选(前台容器全局指针 onMounted+onActivated 升 checklist); 脚本 SubgraphID 容错(未拍板, validator 全局校验已覆盖大半); 搜索/大复合 modal 收进 BaseModal; idea 池(cv-perception · editor-footgun · misc-tools); oxlint 预存 16 错; residue 28 处; StateCycleSmoke 预存红(build.md 在册)。

## 待复核

- 无。(vars.\* 删漏 2026-06-12 用户拍板补删, 已销。)

## 待验证

- 无。(2026-06-12 真机验收全过并销账: 数据层大整理全清单(启动/编辑器/库页/插引用/跑 fishing-v2) + 删被引用模板 referrer 警告旧债。)

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。本节只放**两边都装不下的活上下文**, 现在没有。(2026-06-11 清版: 原"加节点路线图存档指针"等五条已分别沉淀进 docs / 待复核 / build.md, 不再重复。)
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **39** = 28(misc-tools-backlog) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 16 错(oxlint 1.64 新规则, 散在 10 个未涉编辑器的文件)。跑全套测试/检查时按此判红。
