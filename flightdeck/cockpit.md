# Cockpit — YHFish

**Last updated**: 2026-06-17 by 月离 ([yt 脚本控制台 spec](specs/2026-06-17-yt-scripting-console.md)[graduate]: 编辑器内 JS 批量改节点; 拆 P1 子图纳入撤销 + P2 控制台。**经三方 AI 审核逐条评估、采纳要点已加固**(原子语义/overlay/sgID/归一/快照/补全… 评审纪要在案)。设计完成, 待写 P1 plan)。
**Active focus**: **进行中** = [yt 脚本控制台 spec](specs/2026-06-17-yt-scripting-console.md)(graduate)。编辑器内 JS 脚本控制台 (命名空间根 `yt` 对标 blender bpy), 对当前容器主图+子图批量改节点 config。设计已定、spec 已立。**读源码定论**: 编辑器撤销栈 (`useContainerDraft`) 只快照主图 `draft`、子图在 `editorStore` 池里**压根没 undo** → 拆两块: **P1 子图纳入撤销**(核心编辑器, 顺带修既有缺口) → **P2 yt 控制台**(建其上)。**下一步: 写 P1 plan**(见 ## 下一步)。前两个节点小改 (ClickTemplate 重试 / Sleep 默认 1s) 已 land, 真机待验项见 ## 待验证。**默认不 push**。

## 进行中

<!-- AUTO:inprogress -->
- [2026-06-17-yt-scripting-console.md](specs/2026-06-17-yt-scripting-console.md) — 编辑器内 JS 脚本控制台 (命名空间根 yt, 对标 blender bpy): 对当前容器主图+所有子图全节点批量改 config。v1: yt.nodes/selected/container/log + 预留 yt.ops。new Function 执行, set 收集后一次性 applyDraftMutation(一步撤销)+标脏, 抛错零变更, Ctrl+S 落盘。复用 CodeInput(CodeMirror)/walkAllGraphs/PIN_SPECS。
<!-- /AUTO -->

## 下一步

- **写 Part 1 plan** ([yt-scripting-console spec](specs/2026-06-17-yt-scripting-console.md) §分解): 子图纳入编辑器撤销栈 —— 扩展 `useContainerDraft` 的 history 快照/undo/redo 携带**本容器子图**状态, undo/redo 时写回 `editorStore`; 守住复发#5(别误触别的容器编辑器 dirty)。**先精读 `editorStore` + `useContainerDraft` 撤销/快照内部再写 plan**(头号铁律: 核心改动不脑补)。然后 P2 控制台 plan。
- (候选池, P1/P2 之后: 临时窗口抓取 EnumWindows 选窗截图; 复发#5 promotion; idea 池 [cv-perception](specs/cv-perception-pool.md) · [editor-footgun](specs/editor-footgun-backlog.md) · [misc-tools](specs/misc-tools-backlog.md))。

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
