# Cockpit — YHFish

**Last updated**: 2026-06-17 by 月离 (yt 控制台功能完整(实现+全自动门绿, 真机待 smoke); 用户提"三编辑器统一+格式化" → 调研定论 Script/Expr 已共享 EditorModal、控制台是异类 → 立 [unified-code-editor spec](specs/2026-06-17-unified-code-editor.md): 抽 `<CodeEditor>` 共享主体 + 控制台并入 + prettier 懒加载格式化。设计完成待 plan)。
**Active focus**: **两条活线**: ① **[yt 脚本控制台](specs/2026-06-17-yt-scripting-console.md)**(graduate) 实现完整、全自动门绿(执行器12+撤销引擎8+glue4 测 + typecheck/build), **只剩真机 smoke**(见 ## 待验证); ② **[统一代码编辑器](specs/2026-06-17-unified-code-editor.md)** spec 已立(设计完成, 不 graduate): 抽 `<CodeEditor>` 共享主体(从 `EditorModal`)、控制台并入(拿到一致工具栏/参考/补全/折叠)+ 加格式化(prettier 懒加载, JS-only)。读源码定论: Script/Expr **已共享** EditorModal, 控制台是异类 → 只需把控制台并入。**下一步见 ## 下一步**。**默认不 push**。

## 进行中

<!-- AUTO:inprogress -->
- [2026-06-17-unified-code-editor.md](specs/2026-06-17-unified-code-editor.md) — 把 Script/Expr/yt 控制台三个编辑器统一到共享 <CodeEditor> 主体(从 EditorModal 抽出: CodeMirror 视图+工具栏+参考面板+状态栏+折叠/键位); 差异收成 per-mode 配置(extensions builder / 补全源 / hover·参考文档 / commentable·foldable)。EditorModal 重构成壳+确认+<CodeEditor>(保行为等价); YtConsoleModal 改 BaseModal+<CodeEditor mode=yt>+运行/输出。新增格式化 prettier 懒加载(JS-only, Expr N/A)。
- [2026-06-17-yt-scripting-console.md](specs/2026-06-17-yt-scripting-console.md) — 编辑器内 JS 脚本控制台 (命名空间根 yt, 对标 blender bpy): 对当前容器主图+所有子图全节点批量改 config。v1: yt.nodes/selected/container/log + 预留 yt.ops。new Function 执行, set 收集后一次性 applyDraftMutation(一步撤销)+标脏, 抛错零变更, Ctrl+S 落盘。复用 CodeInput(CodeMirror)/walkAllGraphs/PIN_SPECS。
<!-- /AUTO -->

## 下一步

- **统一代码编辑器: review spec → 写 plan → 实现** ([unified-code-editor](specs/2026-06-17-unified-code-editor.md))。**动手前完整读 `EditorModal.vue`** 再无损抽 `<CodeEditor>`(唯一回归点); 控制台改用它 + 格式化(prettier 懒加载)。
- **真机 smoke yt 控制台**(见 ## 待验证)。验过 → flip yt 控制台 spec `done` + graduate 进 `docs/`。
- (候选池, 之后: 临时窗口抓取 EnumWindows 选窗截图; 复发#5 promotion; idea 池 [cv-perception](specs/cv-perception-pool.md) · [editor-footgun](specs/editor-footgun-backlog.md) · [misc-tools](specs/misc-tools-backlog.md))。

## 待复核

- 无。

## 待验证

- ⚠ **ClickTemplate 验证重试 (2026-06-17)** — 真机验: 把会偶尔点空的 ClickTemplate 设 `MaxAttempts=5`、`RetryIntervalMs=500`, 跑一下看点不中时是否自动重点直到模板消失(成功走 Done); 一直点不掉应走 Timeout。单测/build 已绿, 但游戏里实际点击可靠性只能真机验。
- ⚠ **撤销引擎重写 (2026-06-17)** — 真机验普通 Ctrl+Z/Ctrl+Shift+Z 仍正常: 节点增删/改值/拖动(burst 合并一步退)、undo 后再改截断 redo。引擎 8 测全绿 + typecheck, 但 composable 接线没单测(无 test-utils), 真机过一眼稳。
- ⚠ **yt 脚本控制台 (2026-06-17)** — 真机验: 入口 = 工具栏 **⋯ 更多 →「打开 JS 脚本控制台」** (或 Ctrl+K 搜"脚本")。开窗跑 `yt.nodes.filter(n=>n.has('JitterPct')).forEach(n=>n.set('JitterPct',10))` → 报告"改了 N 个", 画布对应节点值变; **一次 Ctrl+Z 全退**(含子图节点); Ctrl+S 落盘。再试个改子图节点的脚本验子图也退得回。执行器/引擎/glue 共 24 测 + build 全绿, 但模态 UI + 真实 applyBulkMutation 接线只能真机验。验过 → flip spec done + graduate。
- (Spec C 真机 smoke + 本会话 chrome/UI 改动 2026-06-15 用户过目无异常, 标记已清。)

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- **UI 系统 (Spec A 设计系统 + Spec B 停靠区/chrome 重组, 均已 done+归档)**: 常驻约定看 [ui.md](checklists/ui.md)(表面分层 / 设置卡片配方 / 配色决策树) + [standalone-window-style](checklists/standalone-window-style.md)(独立工具窗); 共用组件在 `frontend/src/components/common/`(AppCard/StatusPill/SectionHeader/IconBadge/EmptyState/ListRow/AlertBox); 编辑器停靠区在 `frontend/src/components/containers/dock/`。细节查 `archive/specs` 的 Spec A/B 文件。**注意**: Inspector「输出」组已被 Spec C 改成 config.capture 绑定(旧 Spec B 的 VarNameInput 捕获框模型作废)。
- **App 壳重构 (本会话 2026-06-15)**: 侧栏 `AppSidebar` + `useSidebarCollapsed` **已删** → 全局导航全进 `AppTitleBar`(左 品牌+容器/计划 · 右 launcher/设置/关于 + 窗控; 容器保留 jump-back-to-last-container)。chrome/设置/表单页统一 **bordered card 配方**([ui.md](checklists/ui.md) §设置卡片配方; 别用 Inspector 扁平 SectionHeader)。**悬浮窗启动器**模型换 `LauncherGroups`→`LauncherItems []LauncherBlock{type:container|label|hsep|vsep}`(积木式; `FloatingLauncherView` flex-wrap 渲染 + HudShell 图标标题; 旧分组数据弃, 需重建)。中文字体: 等宽栈加微软雅黑/PingFang(中文不回退宋体)。frameless 窗圆角 Win10+wails3 做不了 → [incident](incidents/2026-06-15-frameless-window-rounded-corners-unsupported.md)。
- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **42** = misc-tools-backlog 未翻译 UI(含本分支 chrome/launcher 新增: SettingsLauncher/FloatingLauncherView/HudShell/IconPicker + 1 处 console.log) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 **18** 错(oxlint 1.64 新规则; 2026-06-12 实测修正, 旧账 16 系漂移, 全部已实证存在于 HEAD 同款代码)。跑全套测试/检查时按此判红。
