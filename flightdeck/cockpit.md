# Cockpit — YHFish

**Last updated**: 2026-06-15 by 月离 (Spec B **Part 1 左侧停靠区 code-complete** + 真机反馈修 + 资产面板交互细化: 停靠区落地→smoke 修(落点/拖拽/min660 e6081f1)→按主操作重做资产面板(库/Clip 拖入画布+右键详情 / 模板缩略图网格 c355ce6·52c686a); typecheck/335 test/build:dev 全绿; 真机待用户复 smoke。下一步 Part 2)。
**Active focus**: UI 升级阶段。**Spec A**(地基+门面 4 屏) = done 归档, 设计语言常驻 [ui.md](checklists/ui.md)。**Spec B**(容器编辑器 restyle; vue-flow 画布/节点/连线/pin 只继承 token 不重设计) = **当前焦点**: **Part 1 左侧停靠区 code-complete**(节点库/变量/Snippets/资产 收进 dock, 替 4 modal+2 抽屉; 真机 smoke 待用户), **下一步 Part 2**(Toolbar 三区 + 底部问题条 + Inspector 三态/分组)。**默认不 push**。

## 进行中

<!-- AUTO:inprogress -->
- [2026-06-14-ui-uplift-editor.md](specs/2026-06-14-ui-uplift-editor.md) — "UI 升级第二轮 (Spec B): 容器编辑器 restyle + 布局/UX 重设计。方向 = A 双栏停靠 + Inspector 选中才出 (全量重做, 解 4 大痛点: modal 盖画布 / 三层边栏挤画布 / Inspector 扁平 / Toolbar 主次乱)。① 面板 IA: 左侧自适应宽度停靠区收纳 节点库·变量·Snippets·资产浏览器 (窄列表↔宽网格自适应, 始终挤画布不盖画布), 小弹窗 restyle 留 modal; ② Toolbar 三区分层 (导航 / 主操作 hero / 文档+⋯收纳) + 底部问题条; ③ Inspector 三态收起规则 + SectionHeader 分组。复用 Spec A 设计语言 + 共用组件 (含 AlertBox/SectionHeader/ListRow)。边界: vue-flow 画布·节点框·连线·pin·画布右键 只继承 token, 不重设计。"
<!-- /AUTO -->

**① 真机 smoke Part 1**(task dev 进编辑器: rail 4 图标开/收 · Tab 开收节点库 · 资产 tab 切换+宽态自适应 · 节点字段选模板路由到停靠区 pick 模式 · 双击库项/Clip 插节点; **画布交互不回归**) → **② Part 2 writing-plans**: Toolbar 三区 + 底部问题条 + Inspector 三态/分组(复用 Spec A AlertBox/SectionHeader/ListRow)。Part 3(剩余小弹窗 restyle: 设置/帮助/引用查找/子图脚本预览/新建变量/提升变量 → BaseModal)押后。**边界**: vue-flow 画布/节点/连线/pin/画布右键 只继承 token 不重设计。

其余候选池押后: 临时窗口抓取(EnumWindows 选窗截图); NodeSearchModal/CommandPalette 收 BaseModal; 复发#5 promotion(前台容器全局指针升 checklist); idea 池(cv-perception · editor-footgun · misc-tools)。

## 待复核

- 无。

## 待验证

- ⚠未验证: [archive/plans/2026-06-14-ui-uplift-editor-part1-dock.md](archive/plans/2026-06-14-ui-uplift-editor-part1-dock.md) — 真机 smoke: rail 4 图标开/收 · Tab 开收节点库 · **节点库单击落视口中心 / 拖节点到画布落鼠标点** · 资产 tab 切换+宽态(min660) · 节点字段选模板路由到停靠区; 画布交互不回归。
- ⚠未验证: 资产面板交互细化(2026-06-15 用户拍板 commit c355ce6/52c686a) — **库/Clip 行拖到画布(落松手处)/双击插中心 · 右键·⋯=详情(按需小 modal)·删除 · 列表占满无右栏**; **模板缩略图网格(看图零点击): pick 点图=勾选✓ / 管理 点=选中·⋯·双击=详情·批量**。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- **UI 设计系统 (Spec A 已落地, 常驻 [ui.md](checklists/ui.md) §表面分层 + `style.css`)**: ① 四档表面全派生自 `--ui-bg`, base 钉 neutral-950(比 NuxtUI 默认深一档); 卡片/面板/modal **黑底 + 仅顶部渐隐高光**(不整面提亮, 用户定); 渐变全场只两处(主按钮 `#11c08a→#0a9d6f` + 卡片顶光)。② 7 共用组件在 `frontend/src/components/common/`(AppCard/StatusPill/AlertBox/EmptyState/SectionHeader/IconBadge/ListRow; 有逻辑的带 .helpers.ts 单测)。**Spec B 编辑器屏可消费 AlertBox·SectionHeader·ListRow**(Part 2 门面屏还没用到的 3 个)。③ mono = JetBrains Mono(已打包)。④ **硬约束**: 概念分类色(fuchsia/emerald/amber)+ 日志流身份色(cyan/violet SYS/CTR)是身份色非状态色, 统一散写字面色时跳过。⑤ Spec A brainstorm 贴图在 `.superpowers/brainstorm/146-*`(gitignored)。
- **Spec B Part 1 左侧停靠区已落地** (`frontend/src/components/containers/dock/`): `ContainerEditorDock`(壳 = aside+SplitHandle, 窄300/宽600 双持久化宽度 `editor.dock.narrow|wide`) + `NodeLibraryPanel` + `AssetDockPanel`(UTabs 收 模板/库/Clip 三个 `*AssetPanel`) + `useAssetPicker`(字段→停靠区 pick 通道, 模块单例)。状态 `useSidebarPrefs.leftDrawer` 扩 `'nodes'|'assets'` + `assetTab`。**4 个 explorer modal 已删**, 选模板从节点字段走停靠区 pick 模式(`TemplatePickerField :pin`)。Tab 开收节点库 / 命令面板 navigate.library 跳资产·库 tab。**资产面板交互(2026-06-15 细化)**: 三类按主操作分开 — 库/Clip 主操作=放画布(行可拖到画布 `library-subgraph`/`clip` payload + 双击插中心), 详情(低频)走右键/⋯ → 按需小 modal(复用 `*DetailPanel`), 收掉常开右栏; 模板主操作=看图(`TemplateThumb` 缩略图网格, pick 点图勾选 / 管理 ⋯·双击=详情)。节点库项也可拖到画布(`node-spec`), 单击落视口中心。**Part 2(Toolbar/底条/Inspector)、Part 3(剩余小弹窗 restyle)未做**。
- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **39** = 28(misc-tools-backlog) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 **18** 错(oxlint 1.64 新规则; 2026-06-12 实测修正, 旧账 16 系漂移, 全部已实证存在于 HEAD 同款代码)。跑全套测试/检查时按此判红。
