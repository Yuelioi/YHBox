# Cockpit — YHFish

**Last updated**: 2026-06-17 by 月离 (编辑器统一 + 格式化全落地: 共享 `CodeEditor`(Script/Expr/控制台同体验) + 格式化按钮(prettier 懒加载, 独立 chunk 不进主包, JS-only, Shift+Alt+F)。装 prettier 用 `cd frontend && pnpm add`(绕开 -C bug, 网是通的)。typecheck/97测/i18n/task build 全绿。只剩真机 smoke)。
**Active focus**: **编辑器统一 + 格式化已落地 + 全自动门绿**([unified-code-editor](specs/2026-06-17-unified-code-editor.md)): 抽出共享 `frontend/src/components/expressions/CodeEditor.vue` 主体, `EditorModal`(Script/Expr 放大) + `YtConsoleModal` 都改用它 → 三编辑器 工具栏/参考/补全/折叠/状态栏 一致, 控制台不再简陋; 差异只在 per-mode 配置。**格式化**: prettier 标准版懒加载(独立 chunk 不进主包, 实测 main 891KB 没涨), Script+控制台有「格式化」按钮 + Shift+Alt+F, Expr 无。**另补两项**(用户验收后追加): ① **子图内 Ctrl+Z 撤销** —— `applyDraftMutation` 在子图里编辑时把活动子图也快照进历史(复用 historyEngine sgState), 手动改子图现可撤销; ② **`yt.nodes.` 上下文补全** —— 控制台换上下文感知源 `ytCompletionSource`(yt.→成员 / yt.nodes.·selected.→数组方法 / n.→NodeHandle), 纯判定 `ytCompletionKind` 6 测。typecheck/103测/i18n parity/task build 全绿。**只剩真机 smoke**(见 ## 待验证)。[yt 脚本控制台 spec](specs/2026-06-17-yt-scripting-console.md)(graduate) 逻辑核心(24 测)亦绿、同 smoke。**默认不 push**。

## 进行中

<!-- AUTO:inprogress -->
- [2026-06-17-unified-code-editor.md](specs/2026-06-17-unified-code-editor.md) — 把 Script/Expr/yt 控制台三个编辑器统一到共享 <CodeEditor> 主体(从 EditorModal 抽出: CodeMirror 视图+工具栏+参考面板+状态栏+折叠/键位); 差异收成 per-mode 配置(extensions builder / 补全源 / hover·参考文档 / commentable·foldable)。EditorModal 重构成壳+确认+<CodeEditor>(保行为等价); YtConsoleModal 改 BaseModal+<CodeEditor mode=yt>+运行/输出。新增格式化 prettier 懒加载(JS-only, Expr N/A)。
- [2026-06-17-yt-scripting-console.md](specs/2026-06-17-yt-scripting-console.md) — 编辑器内 JS 脚本控制台 (命名空间根 yt, 对标 blender bpy): 对当前容器主图+所有子图全节点批量改 config。v1: yt.nodes/selected/container/log + 预留 yt.ops。new Function 执行, set 收集后一次性 applyDraftMutation(一步撤销)+标脏, 抛错零变更, Ctrl+S 落盘。复用 CodeInput(CodeMirror)/walkAllGraphs/PIN_SPECS。
<!-- /AUTO -->

## 下一步

- **真机 smoke**(见 ## 待验证): ① Script/Expr 放大编辑无回归(编辑/补全/参考/确认/取消) ② yt 控制台跑通 ③ 格式化(Script/控制台点「格式化」乱缩进 JS 变整齐, Ctrl+Z 一步退)。验过 → unified-editor + yt 控制台两 spec flip done(yt console graduate 进 docs/)。
- (候选池, 之后: 临时窗口抓取 EnumWindows 选窗截图; 复发#5 promotion; idea 池 [cv-perception](specs/cv-perception-pool.md) · [editor-footgun](specs/editor-footgun-backlog.md) · [misc-tools](specs/misc-tools-backlog.md))。

## 待复核

- 无。

## 待验证

- ⚠ **ClickTemplate 验证重试 (2026-06-17)** — 真机验: 把会偶尔点空的 ClickTemplate 设 `MaxAttempts=5`、`RetryIntervalMs=500`, 跑一下看点不中时是否自动重点直到模板消失(成功走 Done); 一直点不掉应走 Timeout。单测/build 已绿, 但游戏里实际点击可靠性只能真机验。
- ⚠ **撤销引擎重写 (2026-06-17)** — 真机验普通 Ctrl+Z/Ctrl+Shift+Z 仍正常: 节点增删/改值/拖动(burst 合并一步退)、undo 后再改截断 redo。引擎 8 测全绿 + typecheck, 但 composable 接线没单测(无 test-utils), 真机过一眼稳。
- ⚠ **yt 脚本控制台 (2026-06-17)** — 真机验: 入口 = 工具栏 **⋯ 更多 →「打开 JS 脚本控制台」** (或 Ctrl+K 搜"脚本")。开窗跑 `yt.nodes.filter(n=>n.has('JitterPct')).forEach(n=>n.set('JitterPct',10))` → 报告"改了 N 个", 画布对应节点值变; **一次 Ctrl+Z 全退**(含子图节点); Ctrl+S 落盘。再试个改子图节点的脚本验子图也退得回。执行器/引擎/glue 共 24 测 + build 全绿, 但模态 UI + 真实 applyBulkMutation 接线只能真机验。验过 → flip spec done + graduate。
- ⚠ **编辑器统一 / EditorModal 重构 (2026-06-17)** — **回归点**: 抽 `CodeEditor` 主体后 `EditorModal`(Script 节点 / Expr 表达式的放大编辑器) 改成壳+`CodeEditor`。真机验 Script/Expr **放大编辑无回归**: 打开放大编辑器 → 编辑/补全/参考面板/折叠/查找/确认回写/取消丢弃 全照旧。另验**格式化**: Script/控制台写乱缩进 JS → 点工具栏「格式化」(或 Shift+Alt+F) → 变整齐(prettier), Ctrl+Z 一步退; Expr 无此按钮。另验**子图内撤销**: 进子图手动改节点 → Ctrl+Z 退回(之前退不回); **控制台补全**: 敲 `yt.` / `yt.nodes.` / `n.` 各有对应提示。typecheck/103测/task build 绿, 但 Vue 组件无单测(无 test-utils), 行为等价靠真机过一眼。
- (Spec C 真机 smoke + 本会话 chrome/UI 改动 2026-06-15 用户过目无异常, 标记已清。)

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- **UI 系统 (Spec A 设计系统 + Spec B 停靠区/chrome 重组, 均已 done+归档)**: 常驻约定看 [ui.md](checklists/ui.md)(表面分层 / 设置卡片配方 / 配色决策树) + [standalone-window-style](checklists/standalone-window-style.md)(独立工具窗); 共用组件在 `frontend/src/components/common/`(AppCard/StatusPill/SectionHeader/IconBadge/EmptyState/ListRow/AlertBox); 编辑器停靠区在 `frontend/src/components/containers/dock/`。细节查 `archive/specs` 的 Spec A/B 文件。**注意**: Inspector「输出」组已被 Spec C 改成 config.capture 绑定(旧 Spec B 的 VarNameInput 捕获框模型作废)。
- **App 壳重构 (本会话 2026-06-15)**: 侧栏 `AppSidebar` + `useSidebarCollapsed` **已删** → 全局导航全进 `AppTitleBar`(左 品牌+容器/计划 · 右 launcher/设置/关于 + 窗控; 容器保留 jump-back-to-last-container)。chrome/设置/表单页统一 **bordered card 配方**([ui.md](checklists/ui.md) §设置卡片配方; 别用 Inspector 扁平 SectionHeader)。**悬浮窗启动器**模型换 `LauncherGroups`→`LauncherItems []LauncherBlock{type:container|label|hsep|vsep}`(积木式; `FloatingLauncherView` flex-wrap 渲染 + HudShell 图标标题; 旧分组数据弃, 需重建)。中文字体: 等宽栈加微软雅黑/PingFang(中文不回退宋体)。frameless 窗圆角 Win10+wails3 做不了 → [incident](incidents/2026-06-15-frameless-window-rounded-corners-unsupported.md)。
- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **42** = misc-tools-backlog 未翻译 UI(含本分支 chrome/launcher 新增: SettingsLauncher/FloatingLauncherView/HudShell/IconPicker + 1 处 console.log) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 **18** 错(oxlint 1.64 新规则; 2026-06-12 实测修正, 旧账 16 系漂移, 全部已实证存在于 HEAD 同款代码)。跑全套测试/检查时按此判红。
