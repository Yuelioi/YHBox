# Cockpit — YHFish

**Last updated**: 2026-06-12 by 月离 (数据层大整理 P1+P2 全部落地: 平铺布局/子图全局化/库 UI 池化/迁移已跑(备份 data.pre-flatten-2026-06-12)。代码全绿, 差真机验收一轮。)
**Active focus**: **数据层大整理代码+迁移全完, 待真机验收**(清单见 待验证)。两个 plan 已归档(P1 57698a2 / P2 766e321+f95afaa+af5bd58)。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

<!-- AUTO:inprogress -->
- [2026-06-12-data-layout-flatten-subgraph-globalize.md](specs/2026-06-12-data-layout-flatten-subgraph-globalize.md) — 数据层大整理 — 目录平铺(删 assets/library 中间层) + 子图全局化(容器只引用不复制) + 架构耐久性根基
<!-- /AUTO -->

## 下一步

真机验收数据层大整理(清单 = 待验证第一条; 删被引用模板 referrer 警告那条旧债顺手一起验)。过了之后: 删 tmp/migrate_flatten.py + tmp/make_testdata.py(一次性工具) + 可删 main.go 的 failIfPreFlattenLayout 防呆闸 + 删备份 bin/data.pre-flatten-2026-06-12(确认无回滚需要后)。spec 待全验过后拍板 flip done。其余候选(无紧迫): 修复 2a0ff140 测试容器的预存悬空引用 sg-0d53b1bb(删那个节点即可); WaitTemplate 孤儿边原子性硬化(真机再现再修); 复发#5 promotion 候选(前台容器全局指针 onMounted+onActivated 升 checklist); 脚本 SubgraphID 容错(未拍板, validator 全局校验已覆盖大半); 搜索/大复合 modal 收进 BaseModal; idea 池(cv-perception · editor-footgun · misc-tools); oxlint 预存 16 错; residue 28 处; StateCycleSmoke 预存红(build.md 在册)。

## 待复核

- 无。(vars.\* 删漏 2026-06-12 用户拍板补删, 已销。)

## 待验证

- ⚠ [archive/plans/2026-06-12-p2-subgraph-globalize.md](archive/plans/2026-06-12-p2-subgraph-globalize.md) — 数据层大整理真机验收: 启动(防呆闸不拦新布局) → 容器列表 → fishing-v2 编辑器(子图解析/双击进) → 库页(列子图+引用计数+复制为新子图) → 子图库选中即插引用+缺变量自动补 → fishing-v2 跑一轮。已知点: 2a0ff140 测试容器报"子图缺失"是源数据预存悬空(sg-0d53b1bb), 非迁移损坏。
- ⚠ [archive/specs/2026-06-11-script-template-dep-extraction.md](archive/specs/2026-06-11-script-template-dep-extraction.md) — 库里删一个被某脚本引用的模板,确认弹「被引用」referrer 警告 + gcBlobs 不回收其 blob(单测已覆盖提取+扫描器接线,差集成/真机这一验)。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。本节只放**两边都装不下的活上下文**, 现在没有。(2026-06-11 清版: 原"加节点路线图存档指针"等五条已分别沉淀进 docs / 待复核 / build.md, 不再重复。)
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **39** = 28(misc-tools-backlog) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 16 错(oxlint 1.64 新规则, 散在 10 个未涉编辑器的文件)。跑全套测试/检查时按此判红。
