# Cockpit — YHFish

**Last updated**: 2026-06-14 by 月离 (Spec A **landed/done 归档**: 四屏真机 smoke 过[关于屏收 `max-w-3xl` 居中列+去卡中卡; v3 base 下沉 neutral-950 + 表面改黑底顶光不整面提亮]; v3 表面系统沉进 [ui.md](checklists/ui.md)。下一步 Spec B 容器编辑器 brainstorm)。
**Active focus**: UI 升级阶段。**Spec A**(设计系统地基 + 主程序门面 4 屏) = **done 已归档**([archive](archive/specs/2026-06-14-ui-uplift-foundation.md); 实现 + 四屏真机 smoke 全过), 设计语言常驻 [ui.md](checklists/ui.md) + `style.css`。**Spec B**(容器编辑器 壳+面板+全部 modal restyle + 布局/UX; vue-flow 画布/节点框/连线/pin 只继承 token 不重设计) = **当前焦点**, 待第二轮 brainstorm + mockup。**默认不 push**。

## 进行中

<!-- AUTO:inprogress -->

<!-- /AUTO -->

## 下一步

**① Spec B 第二轮 brainstorm + mockup**(容器编辑器 restyle + 布局/UX 重设计)。范围: 编辑器壳 + 面板(Toolbar/Breadcrumb/Inspector/左 rail)+ 全部 modal(clip/library/template explorer/子图 props/帮助)的 restyle。**边界(已拍定)**: vue-flow 画布/节点框/连线/pin **只继承 Spec A token, 不做画布级重设计**(避开 incident 高发核心)。复用 Spec A 设计语言 + 7 共用组件(其中 AlertBox·SectionHeader·ListRow 编辑器屏还没消费过), 不重造地基。

其余候选池押后: 临时窗口抓取(EnumWindows 选窗截图); NodeSearchModal/CommandPalette 收 BaseModal; 复发#5 promotion(前台容器全局指针升 checklist); idea 池(cv-perception · editor-footgun · misc-tools)。

## 待复核

- 无。

## 待验证

- 无。(Spec A 四屏真机 smoke 2026-06-14 已过, verify 标记已清。)

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- **UI 设计系统 (Spec A 已落地, 常驻 [ui.md](checklists/ui.md) §表面分层 + `style.css`)**: ① 四档表面全派生自 `--ui-bg`, base 钉 neutral-950(比 NuxtUI 默认深一档); 卡片/面板/modal **黑底 + 仅顶部渐隐高光**(不整面提亮, 用户定); 渐变全场只两处(主按钮 `#11c08a→#0a9d6f` + 卡片顶光)。② 7 共用组件在 `frontend/src/components/common/`(AppCard/StatusPill/AlertBox/EmptyState/SectionHeader/IconBadge/ListRow; 有逻辑的带 .helpers.ts 单测)。**Spec B 编辑器屏可消费 AlertBox·SectionHeader·ListRow**(Part 2 门面屏还没用到的 3 个)。③ mono = JetBrains Mono(已打包)。④ **硬约束**: 概念分类色(fuchsia/emerald/amber)+ 日志流身份色(cyan/violet SYS/CTR)是身份色非状态色, 统一散写字面色时跳过。⑤ Spec A brainstorm 贴图在 `.superpowers/brainstorm/146-*`(gitignored)。
- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **39** = 28(misc-tools-backlog) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 **18** 错(oxlint 1.64 新规则; 2026-06-12 实测修正, 旧账 16 系漂移, 全部已实证存在于 HEAD 同款代码)。跑全套测试/检查时按此判红。
