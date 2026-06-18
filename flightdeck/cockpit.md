# Cockpit — YHFish

**Last updated**: 2026-06-17 by 月离 (编辑器大活线**验收通过 + 收尾**: 三代码编辑器统一(共享 CodeEditor)+ 格式化 + 子图内撤销 + yt 控制台/上下文补全 + modal 不误关, 真机过。unified-editor 设计归档、yt 控制台 graduate 进 [docs](docs/yt-scripting-console.md)。无进行中 spec)。
**Active focus**: **无进行中 spec**。本轮 (2026-06-17) 编辑器大活线**全部验收通过 + 收尾**: ① 三代码编辑器 (Script/Expr/yt 控制台) 统一到共享 `CodeEditor` 主体 + per-mode 配置; ② **格式化** (prettier 懒加载, JS-only); ③ **yt 脚本控制台** (对当前容器批量改节点的 JS 控制台) + 上下文补全; ④ **子图内 Ctrl+Z 撤销**; ⑤ 编辑器 modal 点外部/补全浮层不再误关 (`dismissible=false`)。常驻知识: **[docs/yt-scripting-console](docs/yt-scripting-console.md)** (yt 控制台 + API + 撤销/补全机制); unified-editor 设计在 `archive/specs`。下一步候选见 ## 下一步。**默认不 push**。

## 进行中

<!-- AUTO:inprogress -->
- [2026-06-18-cv-borrow-batch.md](specs/2026-06-18-cv-borrow-batch.md) — 按键精灵 CV 函数表借鉴的 3 个纯 Go 节点: FindColorSignature 颜色签名 / FindTemplateAll 模板全部命中 / DecodeQR 二维码解码; 零新增原生依赖。
- [2026-06-18-cv-borrow-batch.md](plans/2026-06-18-cv-borrow-batch.md) — 实现 cv-borrow-batch spec 的 3 节点: FindColorSignature → DecodeQR → FindTemplateAll, 按风险递增 TDD 落地。
<!-- /AUTO -->

## 下一步

- 无进行中 spec。候选池: 临时窗口抓取(EnumWindows 选窗截图); 复发#5 promotion(前台容器全局指针升 checklist); idea 池([cv-perception](specs/cv-perception-pool.md) · [editor-footgun](specs/editor-footgun-backlog.md) · [misc-tools](specs/misc-tools-backlog.md))。
- (待验证里还挂着 ClickTemplate 验证重试的真机 smoke —— 游戏里跑过再消, 见 ## 待验证。)

## 待复核

- 无。

## 待验证

- ⚠ **ClickTemplate 验证重试 (2026-06-17)** — 真机验: 把会偶尔点空的 ClickTemplate 设 `MaxAttempts=5`、`RetryIntervalMs=500`, 跑一下看点不中时是否自动重点直到模板消失(成功走 Done); 一直点不掉应走 Timeout。单测/build 已绿, 但游戏里实际点击可靠性只能真机验。
- (编辑器大活线 2026-06-17 用户验收通过, 标记已清: 三编辑器统一 / 格式化 / 子图内 Ctrl+Z 撤销 / yt 控制台 + 补全 / modal 不误关 / EditorModal 重构无回归。常驻知识 → [docs/yt-scripting-console](docs/yt-scripting-console.md) + archive/specs。)
- (Spec C 真机 smoke + 本会话 chrome/UI 改动 2026-06-15 用户过目无异常, 标记已清。)

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- **UI 系统 (Spec A 设计系统 + Spec B 停靠区/chrome 重组, 均已 done+归档)**: 常驻约定看 [ui.md](checklists/ui.md)(表面分层 / 设置卡片配方 / 配色决策树) + [standalone-window-style](checklists/standalone-window-style.md)(独立工具窗); 共用组件在 `frontend/src/components/common/`(AppCard/StatusPill/SectionHeader/IconBadge/EmptyState/ListRow/AlertBox); 编辑器停靠区在 `frontend/src/components/containers/dock/`。细节查 `archive/specs` 的 Spec A/B 文件。**注意**: Inspector「输出」组已被 Spec C 改成 config.capture 绑定(旧 Spec B 的 VarNameInput 捕获框模型作废)。
- **App 壳重构 (本会话 2026-06-15)**: 侧栏 `AppSidebar` + `useSidebarCollapsed` **已删** → 全局导航全进 `AppTitleBar`(左 品牌+容器/计划 · 右 launcher/设置/关于 + 窗控; 容器保留 jump-back-to-last-container)。chrome/设置/表单页统一 **bordered card 配方**([ui.md](checklists/ui.md) §设置卡片配方; 别用 Inspector 扁平 SectionHeader)。**悬浮窗启动器**模型换 `LauncherGroups`→`LauncherItems []LauncherBlock{type:container|label|hsep|vsep}`(积木式; `FloatingLauncherView` flex-wrap 渲染 + HudShell 图标标题; 旧分组数据弃, 需重建)。中文字体: 等宽栈加微软雅黑/PingFang(中文不回退宋体)。frameless 窗圆角 Win10+wails3 做不了 → [incident](incidents/2026-06-15-frameless-window-rounded-corners-unsupported.md)。
- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system; **yt 控制台 + 代码编辑器 → yt-scripting-console**); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。
- **代码编辑器统一 (2026-06-17)**: 三个编辑器 (Script 节点 / Expr 表达式 / yt 控制台) 共享 `frontend/src/components/expressions/CodeEditor.vue` 主体 + per-mode 配置 (extensions builder / 补全源 / 参考文档 / commentable·foldable·formattable); 放大壳 = `EditorModal`, 控制台 = `YtConsoleModal`。编辑器 modal `dismissible=false`(补全浮层挂 body 防点外部误关)。撤销引擎 `historyEngine`(纯, 历史条目带 sgState → 主图+子图统一撤销)。细节 [docs/yt-scripting-console](docs/yt-scripting-console.md)。
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **42** = misc-tools-backlog 未翻译 UI(含本分支 chrome/launcher 新增: SettingsLauncher/FloatingLauncherView/HudShell/IconPicker + 1 处 console.log) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 **18** 错(oxlint 1.64 新规则; 2026-06-12 实测修正, 旧账 16 系漂移, 全部已实证存在于 HEAD 同款代码)。跑全套测试/检查时按此判红。
