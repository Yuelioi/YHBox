# Cockpit — YHFish

**Last updated**: 2026-06-15 by 月离 (**Spec C 立项** — 输出自动捕获 + Inspector 输出组统一绑定; brainstorm 拍定、spec 已写, 待用户过目 → 转 plan。UI 升级 Spec A+B 已收官归档)。
**Active focus**: **Spec C — 输出自动捕获** ([spec](specs/2026-06-15-output-auto-capture.md))。框架 dispatch 自动把出口产出写进绑定变量 (替"逐节点手声明捕获框 + node.Capture()") + 前端「输出」组统一绑定 UI (方案 A: 按钮绑+chip)。所有执行节点产出自动可绑。UI 升级 (Spec A+B) 已 done 归档。**默认不 push**。

## 进行中

<!-- AUTO:inprogress -->
- [2026-06-15-output-auto-capture.md](specs/2026-06-15-output-auto-capture.md) — "输出自动捕获 + Inspector 输出组统一绑定 (Spec C)。取消'逐节点手声明 Semantic:capture 输入框 + Run 里手调 node.Capture()'这套, 改成框架在 dispatch routeResult 把 fire 出口的 OutputData[字段] 自动写进用户绑定的变量。前端 Inspector「输出」组合并掉 Part 2 的'速览'+'捕获'两套, 每个可绑产出一行: 翻译名 + 类型 + 「+绑变量」按钮 (绑后显 → \$var ✕ chip), 写 config.capture{字段:变量名}。所有执行节点的数据产出自动可绑 (含现在漏声明捕获框的 PlayClip.Error/Code)。核心不变量: 被捕获值必须是出口 Data 字段 (从 OutputData 读) —— 模板三件套的 Found 布尔补成显式 Data 字段。删 13 文件 27 个 capture 输入 + node.Capture 助手。迁移条件化 (旧 config.literal[Capture<X>] → config.capture[<X>], 没有就跳过)。边界: 不碰 vue-flow 画布/节点/连线/pin, 绑定全在 Inspector。"
<!-- /AUTO -->

**Spec C 进行中**: brainstorm done、spec 已写并待用户过目 → 过了转 writing-plans。三件设计已拍: ① 框架自动捕获 (dispatch `routeResult` 读 `result.OutputData` 写绑定变量) ② 前端输出组方案 A (按钮绑+chip, 翻译统一) ③ 模板三件套 Found 布尔补成显式 Data 字段 (核心不变量: 被捕获值必须是出口 Data 字段)。迁移条件化。

候选池(Spec C 后): 临时窗口抓取(EnumWindows 选窗截图); 复发#5 promotion(前台容器全局指针升 checklist); idea 池(cv-perception · editor-footgun · misc-tools)。

## 待复核

- 无。

## 待验证

- 无。(Spec B Part 1 + Part 2 chrome 重组 真机 smoke 2026-06-15 用户均验过, verify 标记已清。)

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- **UI 设计系统 (Spec A 已落地, 常驻 [ui.md](checklists/ui.md) §表面分层 + `style.css`)**: ① 四档表面全派生自 `--ui-bg`, base 钉 neutral-950(比 NuxtUI 默认深一档); 卡片/面板/modal **黑底 + 仅顶部渐隐高光**(不整面提亮, 用户定); 渐变全场只两处(主按钮 `#11c08a→#0a9d6f` + 卡片顶光)。② 7 共用组件在 `frontend/src/components/common/`(AppCard/StatusPill/AlertBox/EmptyState/SectionHeader/IconBadge/ListRow; 有逻辑的带 .helpers.ts 单测)。**Spec B Part 2 已消费 SectionHeader**(Inspector 三组); AlertBox·ListRow 仍备用(Part 2 问题条沿用 modal 同款 tint 边框配方, 没强行套, 二号铁律不为用而用); 主按钮绿渐变 `btn-primary-raised` 由 `vite.config` button.compoundVariants 自动套到 `primary+solid`(运行 hero 直接用)。③ mono = JetBrains Mono(已打包)。④ **硬约束**: 概念分类色(fuchsia/emerald/amber)+ 日志流身份色(cyan/violet SYS/CTR)是身份色非状态色, 统一散写字面色时跳过。⑤ Spec A brainstorm 贴图在 `.superpowers/brainstorm/146-*`(gitignored)。
- **Spec B Part 1 左侧停靠区已落地** (`frontend/src/components/containers/dock/`, done 真机过): `ContainerEditorDock`(壳 = aside+SplitHandle, 窄 default300/min240 · 宽 default520/min450 双持久化宽度 `editor.dock.narrow|wide`) + `NodeLibraryPanel` + `AssetDockPanel`(UTabs 收 模板/库/Clip 三个 `*AssetPanel`) + `AssetSelectionBar`(批量上下文条) + `TemplateThumb` + `useAssetPicker`(字段→停靠区 pick 通道, 模块单例)。状态 `useSidebarPrefs.leftDrawer` 扩 `'nodes'|'assets'` + `assetTab`。**4 个 explorer modal 已删**, 选模板从节点字段走停靠区 pick 模式(`TemplatePickerField :pin`)。Tab 开收节点库 / 命令面板 navigate.library 跳资产·库 tab。**资产面板交互(终态)**: 三类统一 **单击选中 · 双击详情 · 拖拽插画布**(库/Clip=`library-subgraph`/`clip` payload, 节点库=`node-spec` 单击落视口中心); 详情=按需小 modal(复用 `*DetailPanel`, 去常开右栏); 批量=顶部 `AssetSelectionBar`(选中才出)+ 底部仅分页; 模板=缩略图网格(pick 点图勾选 ✓)。
- **Spec B Part 2 chrome 重组已落地** (done 真机过): ① **NodeInspector** 扁平 section → `SectionHeader` 三组「基础/输入/输出」; **输出组 = 输出捕获(`VarNameInput` 绑变量, 默认展开)+ `pinsFor` 出口 pin 速览(只读)** —— 捕获绑变量归输出组, 别归输入组(9d66558 修过一次回归); header 不动。② **Inspector 三态** = `composables/editor/inspectorMode.ts` 纯函数 `resolveInspectorMode`(node/subgraph/collapsed) + `showInspector` computed; 折叠 ⊟ 从 toolbar 挪到**画布右边缘单一 toggle**(collapsed 态不显); 根图空选自动收起画布全宽; 容器概览+热键 → `ContainerOverviewPopover`(toolbar 左区面包屑旁), 快捷开始 → `CanvasEmptyState`(画布空时)。③ **Toolbar 三区** = 左(返回·面包屑·概览·撤销/重做) / 中(录制 neutral + 运行 hero primary-solid) / 右(校验·保存带 dirty 黄点·自动布局下拉·⋯); 设置进 ⋯、自动布局出 ⋯。④ **底部问题条** = `EditorProblemsBar`(收起状态徽章/展开列错误+跳转/修复, 挤画布不盖) **替掉 `ValidationErrorPanel` modal(已删)**; `composables/editor/problemsBar.ts` 纯函数 `summarizeProblems` + 单测; view 状态 `validationPanelOpen`→`problemsExpanded`+`validationRan`。⑤ **Part 3(浮层 modal restyle)= 无代码改动**: §5 列的 6 弹窗在 Spec A 建 BaseModal 时已全迁; CommandPalette/NodeSearchModal 按 §5 维持自有 UModal 壳、token 干净; 详情面板 Part 1 已处理。Spec B 收官归档。
- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **39** = 28(misc-tools-backlog) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 **18** 错(oxlint 1.64 新规则; 2026-06-12 实测修正, 旧账 16 系漂移, 全部已实证存在于 HEAD 同款代码)。跑全套测试/检查时按此判红。
