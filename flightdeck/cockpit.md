# Cockpit — YHFish

**Last updated**: 2026-06-15 by 月离 (**Spec C Part 1 后端自动捕获 done** — config.capture 存储 + 路径①routeResult 自动捕获 + 路径②ctx.CaptureOutput(Loop/ForEach) + 删 13 文件 capture 输入/助手 + validator + BindableFields。go build/test/vet 绿(余 10 预存 fish fixture 缺文件失败, 非回归)。下一步 Part 2 前端)。
**Active focus**: **Spec C — 输出自动捕获**([spec](specs/2026-06-15-output-auto-capture.md))。**Part 1 后端 done**([plan](plans/2026-06-15-output-auto-capture-part1-backend.md), commits 816d4cb..89e16c1)。**下一步 Part 2 前端**(Inspector「输出」组方案 A 按钮绑+chip + 翻译 i18n + useVarMutations 消费者审计改读 config.capture)。**落地精度#7: P1+P2 一个发布单元**(中间态编辑器输出捕获 UI 暂不可用)。UI 升级 (Spec A+B) 已归档。**默认不 push**。

## 进行中

<!-- AUTO:inprogress -->
- [2026-06-15-output-auto-capture.md](specs/2026-06-15-output-auto-capture.md) — "输出自动捕获 + Inspector 输出组统一绑定 (Spec C)。取消'逐节点手声明 Semantic:capture 输入框'这套, 捕获绑定改存 config.capture{字段:变量名}, 前端 Inspector「输出」组统一一行式绑定 (方案 A: 按钮绑+chip, 翻译统一)。**两条写路径** (节点形态决定, 非统一): ① fire-time 自动捕获 — 出口 Data 字段值在 dispatch routeResult 由框架自动写进绑定变量 (~11 个检测/截图/脚本节点, 零节点代码); ② region per-iteration 显式捕获 — Loop/ForEach 的 Index/Item 在 RunRegion 每轮由节点调 helper 读 config.capture 写 (不经 routeResult)。模板三件套 Found 布尔补成显式 Data 字段。**消费者审计** (config.capture 是新 var-ref 站): useVarMutations 5 处 (rename/count/listUsageNodeIDs/deleteVar-cascade/listUsageRefs) + 后端 validator + referrers 全改读 config.capture。迁移条件化 + per-node 映射。边界: 不碰 vue-flow 画布/节点/连线/pin。"
<!-- /AUTO -->

**Spec C Part 1 后端 done**(commits 816d4cb..89e16c1)。落地: `config.capture{字段:变量名}` 存储; **路径① fire-time**(`ContainerRunner.applyCaptures` 钩 `dispatch_v5.routeResult` 成功路径 `result.OutputData` + 失败路径 `{Error,Code}`, `r.bundle.Vars.SetScoped` auto); **路径② region**(`ctx.CaptureOutput(field,value)` + `ctxImpl.captureBindings`, Loop.Index / ForEach.Item+Index 声明为 Body 出口 Data 字段, RunRegion 每轮调); 模板三件套 `Matched` bool Data 字段(两出口 Set, 命名避与 Found 出口冲突); 删 13 文件 27 capture 输入 + 11 `node.Capture` + 助手 + `InputSpec.CaptureType`; `node.BindableFields`(单一来源)+ `validateCaptureRefs`(var-ref→INVALID_VAR_REF / 字段→INVALID_PIN)。**遗留 Part 2**: `internal/catalog/node-i18n.json` stale capture labels 待重生成; 编辑器输出捕获 UI 暂不可用(P1+P2 一个发布单元)。

**下一步 Part 2(前端)**: Inspector「输出」组方案 A(按钮绑+chip, 翻译统一 `node.<kind>.output.<字段>`); **消费者审计**(落地精度#2): `useVarMutations` 5 处(rename/count/listUsageNodeIDs/deleteVar-cascade/listUsageRefs)改读 `config.capture`(漏改=悬空, [[2026-05-29-storage-convention-consumer-audit-gap]]); 删旧 captureLiterals UI。Part 3: 迁移(条件)+ 真机 smoke。

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
