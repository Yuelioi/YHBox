# Cockpit — YHFish

**Last updated**: 2026-06-17 by 月离 (本会话: ClickTemplate 加验证重试 — 新 pin MaxAttempts(默认1=旧行为)/RetryIntervalMs(默认500); 点完查模板消失没、没消失就重点, 点满还在走 Timeout(Matched 区分没出现/没点掉)。治偶发"点了但模板还在"。typecheck/go test/task build 全绿)。
**Active focus**: 无进行中 spec。**本会话 (2026-06-17)**: ClickTemplate「点完验证+没消失重点」(MaxAttempts/RetryIntervalMs), 知识落 [incident 2026-06-13](incidents/2026-06-13-template-click-fires-on-first-match-too-early.md) 续章(第二种"点了不生效"= 没点中, 跟 SettleMs 正交); **真机待用户验**(见 ## 待验证)。上一增量 Spec C + chrome 重构均已 done/归档/过目(细节见 ## 关键上下文)。下一步候选见 ## 下一步。**默认不 push**。

## 进行中

<!-- AUTO:inprogress -->

<!-- /AUTO -->

## 下一步

- 无进行中 spec。候选池: 临时窗口抓取(EnumWindows 选窗截图); 复发#5 promotion(前台容器全局指针升 checklist); idea 池([cv-perception](specs/cv-perception-pool.md) · [editor-footgun](specs/editor-footgun-backlog.md) · [misc-tools](specs/misc-tools-backlog.md))。

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
