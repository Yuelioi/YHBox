# Cockpit — YHFish

**Last updated**: 2026-06-17 by 月离 (yt 控制台续: 落地撤销引擎 `historyEngine`(纯, 8 测) + `useContainerDraft.applyBulkMutation`(主图+触及子图落一条可撤销条目, 子图 undo/redo round-trip 验过); 加之前纯执行器, **逻辑核心齐了**。typecheck + 195 测全绿)。
**Active focus**: **进行中** = [yt 脚本控制台 spec](specs/2026-06-17-yt-scripting-console.md)(graduate)。编辑器内 JS 批量改节点 (命名空间根 `yt`)。范围: 子图全局/手动改不走撤销 → 只做经控制台批量改的**有界一步撤销**(见 spec §撤销机制)。**逻辑核心已落地+测全绿**: ① 纯执行器 `runConsoleScript`(`src/lib/ytConsole`, 12 测); ② 撤销引擎 `historyEngine`(纯, 8 测, 含子图批量改 undo/redo round-trip) + `useContainerDraft.applyBulkMutation`(主图+触及子图一条可撤销条目, 引擎薄封装)。**下一步 = UI 装配 + glue**(见 ## 下一步)。前两个节点小改已 land, 真机待验见 ## 待验证。**默认不 push**。

## 进行中

<!-- AUTO:inprogress -->
- [2026-06-17-yt-scripting-console.md](specs/2026-06-17-yt-scripting-console.md) — 编辑器内 JS 脚本控制台 (命名空间根 yt, 对标 blender bpy): 对当前容器主图+所有子图全节点批量改 config。v1: yt.nodes/selected/container/log + 预留 yt.ops。new Function 执行, set 收集后一次性 applyDraftMutation(一步撤销)+标脏, 抛错零变更, Ctrl+S 落盘。复用 CodeInput(CodeMirror)/walkAllGraphs/PIN_SPECS。
<!-- /AUTO -->

## 下一步

- **yt 控制台 UI + glue** (逻辑核心 执行器+撤销引擎 已绿; spec §UI): 控制台**模态**(复用 `CodeInput`) + Ctrl+K 命令面板入口(i18n `editor.jsConsole.*`) + `yt.*` 静态补全 + **glue**: draft+子图+`PIN_SPECS`/`KIND_DEFAULTS` 组装 `NodeModel[]` → `runConsoleScript` → 按 sgID 分组 `applied` → `applyBulkMutation` 落地(主图写 draft / 子图写 editorStore) → 渲染报告。**前端 vitest 跑法见 [build.md](checklists/build.md) §前端单测**(别用 `pnpm -C frontend test`)。
- (候选池, 本功能之后: 临时窗口抓取 EnumWindows 选窗截图; 复发#5 promotion; idea 池 [cv-perception](specs/cv-perception-pool.md) · [editor-footgun](specs/editor-footgun-backlog.md) · [misc-tools](specs/misc-tools-backlog.md))。

## 待复核

- 无。

## 待验证

- ⚠ **ClickTemplate 验证重试 (2026-06-17)** — 真机验: 把会偶尔点空的 ClickTemplate 设 `MaxAttempts=5`、`RetryIntervalMs=500`, 跑一下看点不中时是否自动重点直到模板消失(成功走 Done); 一直点不掉应走 Timeout。单测/build 已绿, 但游戏里实际点击可靠性只能真机验。
- (Spec C 真机 smoke + 本会话 chrome/UI 改动 2026-06-15 用户过目无异常, 标记已清。)

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- **UI 系统 (Spec A 设计系统 + Spec B 停靠区/chrome 重组, 均已 done+归档)**: 常驻约定看 [ui.md](checklists/ui.md)(表面分层 / 设置卡片配方 / 配色决策树) + [standalone-window-style](checklists/standalone-window-style.md)(独立工具窗); 共用组件在 `frontend/src/components/common/`(AppCard/StatusPill/SectionHeader/IconBadge/EmptyState/ListRow/AlertBox); 编辑器停靠区在 `frontend/src/components/containers/dock/`。细节查 `archive/specs` 的 Spec A/B 文件。**注意**: Inspector「输出」组已被 Spec C 改成 config.capture 绑定(旧 Spec B 的 VarNameInput 捕获框模型作废)。
- **App 壳重构 (本会话 2026-06-15)**: 侧栏 `AppSidebar` + `useSidebarCollapsed` **已删** → 全局导航全进 `AppTitleBar`(左 品牌+容器/计划 · 右 launcher/设置/关于 + 窗控; 容器保留 jump-back-to-last-container)。chrome/设置/表单页统一 **bordered card 配方**([ui.md](checklists/ui.md) §设置卡片配方; 别用 Inspector 扁平 SectionHeader)。**悬浮窗启动器**模型换 `LauncherGroups`→`LauncherItems []LauncherBlock{type:container|label|hsep|vsep}`(积木式; `FloatingLauncherView` flex-wrap 渲染 + HudShell 图标标题; 旧分组数据弃, 需重建)。中文字体: 等宽栈加微软雅黑/PingFang(中文不回退宋体)。frameless 窗圆角 Win10+wails3 做不了 → [incident](incidents/2026-06-15-frameless-window-rounded-corners-unsupported.md)。
- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **42** = misc-tools-backlog 未翻译 UI(含本分支 chrome/launcher 新增: SettingsLauncher/FloatingLauncherView/HudShell/IconPicker + 1 处 console.log) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 **18** 错(oxlint 1.64 新规则; 2026-06-12 实测修正, 旧账 16 系漂移, 全部已实证存在于 HEAD 同款代码)。跑全套测试/检查时按此判红。
