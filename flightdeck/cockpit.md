# Cockpit — YHFish

**Last updated**: 2026-06-14 by 月离 (UI 升级 Spec A: Part 1 已落地归档(地基 + 7 组件, 全绿); **Part 2 逐屏迁移 plan 已写**(P1 容器列表 / P2 计划 / P3 关于 / P4 主壳轻触, 消费 4 组件), 待选执行模式开工)。
**Active focus**: UI 升级阶段 (样式统一 + 工具库 + 样式升级)。**Spec A** = 设计系统地基 + 主程序门面 (外壳/容器列表/计划/关于/LogPanel)，[spec](specs/2026-06-14-ui-uplift-foundation.md) 在做。**Part 1**(地基 token/主按钮渐变/子图坐标常量 + 7 共用组件 AppCard·AlertBox·EmptyState·StatusPill·SectionHeader·IconBadge·ListRow) **已落地归档** [archive](archive/plans/2026-06-14-ui-uplift-foundation-plan.md)。**Part 2**(逐屏迁移 P1-P4) [plan 已写](plans/2026-06-14-ui-uplift-migration-plan.md), 待执行(消费 AppCard/StatusPill/EmptyState/IconBadge 4 个; AlertBox/SectionHeader/ListRow 留 Spec B)。**Spec B**(容器编辑器 壳+面板+全部 modal 的 restyle + 布局/UX; vue-flow 画布/节点框/连线/pin 只继承 token 不重设计) = 第二轮 brainstorm+mockup。**默认不 push**。

## 进行中

<!-- AUTO:inprogress -->
- [2026-06-14-ui-uplift-foundation.md](specs/2026-06-14-ui-uplift-foundation.md) — "UI 升级第一轮：设计系统地基(v3 克制精致 token + 字体角色) + 共用组件(AppCard/AlertBox/EmptyState/StatusPill/SectionHeader/IconBadge/ListRow) + 主程序门面屏 restyle(容器列表/计划/关于/LogPanel/侧栏) + 工具库常量收敛；编辑器画布与重设计在 Spec B"
- [2026-06-14-ui-uplift-migration-plan.md](plans/2026-06-14-ui-uplift-migration-plan.md) — "Spec A 实现计划 Part 2：逐屏迁移到 Part 1 共用组件 + v3 token —— P1 容器列表(AppCard/StatusPill/EmptyState/mono/固定列 + 删除 modal 收 useConfirm) · P2 计划(EmptyState/StatusPill/mono) · P3 关于(AppCard×5/IconBadge avatar/mono) · P4 主壳轻触(AppSidebar 绿 active tint / LogPanel filter token / AppStatusBar 验不动)。迁移即升级、删旧散写样式; 离屏视觉自检 + 真机过"
<!-- /AUTO -->

## 下一步

**执行 [Part 2 plan](plans/2026-06-14-ui-uplift-migration-plan.md) — Task 1 起**: i18n 新键 → P1 容器列表(卡片 AppCard+StatusPill+mono+固定列 / 空状态 EmptyState / 单删收 useConfirm) → P2 计划(EmptyState+StatusPill+mono) → P3 关于(AppCard×5+IconBadge avatar+mono) → P4 主壳轻触(AppSidebar 绿 active / LogPanel filter) → 离屏视觉自检(真实屏灌 mock 数据)+ 全量门禁 + 真机。执行模式待用户选(Part 1 用的 subagent-driven)。

Part 2 落地后 Spec A 全完 → flip spec done + 归档 spec & 两 plan。然后 Spec B(编辑器)第二轮 brainstorm+mockup。其余候选池押后: 临时窗口抓取(EnumWindows 选窗截图); NodeSearchModal/CommandPalette 收 BaseModal; 复发#5 promotion(前台容器全局指针升 checklist); idea 池(cv-perception · editor-footgun · misc-tools)。

## 待复核

- 无。

## 待验证

- 无。(editor-rail / 校验问题面板 2026-06-12 真机过, 已销; 详见各 archive spec verified 字段。)

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- **UI 升级活上下文 (Spec A; Part 1 已交付)**: ① v3 设计语言**已落地** — `raised-surface`/`overlay-surface` `@utility` + `.btn-primary-raised` 主按钮渐变(`#11c08a→#0a9d6f`)在 `style.css`; vite.config `button.compoundVariants` 注入主按钮; **渐变全场只两处**(主按钮 + 卡片顶光)。7 共用组件在 `frontend/src/components/common/`(AppCard/StatusPill/AlertBox/EmptyState/SectionHeader/IconBadge/ListRow; 有逻辑的带 .helpers.ts 单测), Part 2 逐屏消费。视觉自检过(观感克制、`hover:raised-surface` 实证生效 → 见 [ui.md](checklists/ui.md))。② mono 字体**定了用当前 JetBrains Mono**(已打包; Part 1 不碰字体族沿用全局)。③ 硬约束: 概念分类色(About fuchsia/emerald/amber)+ 日志流身份色(LogPanel cyan/violet SYS/CTR)是身份色非状态色, **统一散写字面色时跳过**。④ brainstorm 可视化贴图在 `.superpowers/brainstorm/146-*`(gitignored, 服务 auto-exit)。
- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **39** = 28(misc-tools-backlog) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 **18** 错(oxlint 1.64 新规则; 2026-06-12 实测修正, 旧账 16 系漂移, 全部已实证存在于 HEAD 同款代码)。跑全套测试/检查时按此判红。
