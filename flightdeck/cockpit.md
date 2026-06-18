# Cockpit — YHFish

**Last updated**: 2026-06-18 by 月离 (CV 低依赖借鉴批 **3 节点 ship**: FindColorSignature / DecodeQR / FindTemplateAll, 自动化全绿 + 生产 exe 构建过, **真机 smoke 待验**。修了根因: 捕获模型 Spec C config.capture 而非已删的 Capture 框, 顺手修正 add-node.md/node-data-flow.md。spec/plan 归档)。
**Active focus**: **无进行中 spec**。本轮 (2026-06-18) CV 低依赖借鉴批落地 (按键精灵 CV 函数表借鉴, 全纯 Go): ① **FindColorSignature** 颜色签名(锚点+N偏移点扫描); ② **DecodeQR** 二维码(引 gozxing 纯 Go); ③ **FindTemplateAll** 模板全部命中(`pkg/vision.MatchAll` 3×3极大+NMS, 与 `Match` 共享 `correlationMap`; `TemplateMatcher.DetectAll` 4 实现)。自动化全绿(含 `Match==MatchAll` 等价性)+ `task build` 过, **真机 smoke 待验** (见 ## 待验证)。spec/plan 已归档 `archive/`。**默认不 push**。

## 进行中

<!-- AUTO:inprogress -->

<!-- /AUTO -->

## 下一步

- **先真机 smoke 验 CV 借鉴批** (见 ## 待验证) —— 三节点拖出来各跑一遍, 过了这批彻底收。
- 无进行中 spec。候选池: 临时窗口抓取(EnumWindows 选窗截图); 复发#5 promotion(前台容器全局指针升 checklist); **cv-perception 池剩余** (ONNX/YOLO/OCR/blob 等大依赖路线, 本批只挑了低依赖三件, [cv-perception](specs/cv-perception-pool.md)); idea 池([editor-footgun](specs/editor-footgun-backlog.md) · [misc-tools](specs/misc-tools-backlog.md))。
- (待验证里还挂着 ClickTemplate 验证重试的真机 smoke —— 游戏里跑过再消, 见 ## 待验证。)

## 待复核

- 无。

## 待验证

- ⚠ **CV 借鉴批真机 smoke (2026-06-18)** — 三节点(颜色签名 / 二维码解码 / 模板全部命中)在侧栏/右键/explorer 可加并各跑一遍: 颜色签名屏幕拾色验命中; QR 摆屏读对内容; FindTemplateAll 同图标铺 N 个验 Count/Matches/Primary。自动化全绿(含 `Match==MatchAll` 等价性 + 生产 exe 构建), 但生产 matcher `templateMatcherAdapter.DetectAll`(无单测)端到端只此一处验。详见 [archive/plans/2026-06-18-cv-borrow-batch](archive/plans/2026-06-18-cv-borrow-batch.md) 的 verify。
- ⚠ **ClickTemplate 验证重试 (2026-06-17)** — 真机验: 把会偶尔点空的 ClickTemplate 设 `MaxAttempts=5`、`RetryIntervalMs=500`, 跑一下看点不中时是否自动重点直到模板消失(成功走 Done); 一直点不掉应走 Timeout。单测/build 已绿, 但游戏里实际点击可靠性只能真机验。
- (编辑器大活线 2026-06-17 用户验收通过, 标记已清: 三编辑器统一 / 格式化 / 子图内 Ctrl+Z 撤销 / yt 控制台 + 补全 / modal 不误关 / EditorModal 重构无回归。常驻知识 → [docs/yt-scripting-console](docs/yt-scripting-console.md) + archive/specs。)
- (Spec C 真机 smoke + 本会话 chrome/UI 改动 2026-06-15 用户过目无异常, 标记已清。)

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- **UI 系统 (Spec A 设计系统 + Spec B 停靠区/chrome 重组, 均已 done+归档)**: 常驻约定看 [ui.md](checklists/ui.md)(表面分层 / 设置卡片配方 / 配色决策树) + [standalone-window-style](checklists/standalone-window-style.md)(独立工具窗); 共用组件在 `frontend/src/components/common/`(AppCard/StatusPill/SectionHeader/IconBadge/EmptyState/ListRow/AlertBox); 编辑器停靠区在 `frontend/src/components/containers/dock/`。细节查 `archive/specs` 的 Spec A/B 文件。**注意**: Inspector「输出」组已被 Spec C 改成 config.capture 绑定(旧 Spec B 的 VarNameInput 捕获框模型作废)。
- **App 壳重构 (本会话 2026-06-15)**: 侧栏 `AppSidebar` + `useSidebarCollapsed` **已删** → 全局导航全进 `AppTitleBar`(左 品牌+容器/计划 · 右 launcher/设置/关于 + 窗控; 容器保留 jump-back-to-last-container)。chrome/设置/表单页统一 **bordered card 配方**([ui.md](checklists/ui.md) §设置卡片配方; 别用 Inspector 扁平 SectionHeader)。**悬浮窗启动器**模型换 `LauncherGroups`→`LauncherItems []LauncherBlock{type:container|label|hsep|vsep}`(积木式; `FloatingLauncherView` flex-wrap 渲染 + HudShell 图标标题; 旧分组数据弃, 需重建)。中文字体: 等宽栈加微软雅黑/PingFang(中文不回退宋体)。frameless 窗圆角 Win10+wails3 做不了 → [incident](incidents/2026-06-15-frameless-window-rounded-corners-unsupported.md)。
- **CV 检测节点扩充 (2026-06-18, 已归档)**: `internal/nodes/detect/` 新增 **FindColorSignature**(颜色签名)/ **DecodeQR**(QR, 引 `gozxing` 纯 Go)/ **FindTemplateAll**(模板全部命中)。纯算法在 `pkg/vision/`(`color_signature`/`qr`/`match_all`); `MatchAll`(3×3极大+NMS)与 `Match` 共享 `correlationMap`。`VisionService`(+adapter+stub) 加 FindColorSignature/DecodeQR/MatchAll; `TemplateMatcher` 加 `DetectAll`(4 实现, 生产=`wire_container.go:templateMatcherAdapter`)。设计/评审记录 [archive/specs/2026-06-18-cv-borrow-batch](archive/specs/2026-06-18-cv-borrow-batch.md)。⚠ **捕获模型铁律**(本轮根因坑): 产出节点**只声明 `OutputSpec.Data` + `.Set()`, 不加 `Capture<字段>` 框、不调 `node.Capture`**(Spec C 早删) —— 范式 `detect_color_blobs.go`, 已在 [add-node.md §1b](checklists/add-node.md) + [node-data-flow](checklists/2026-06-05-node-data-flow.md) 修正。
- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system; **yt 控制台 + 代码编辑器 → yt-scripting-console**); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。
- **代码编辑器统一 (2026-06-17)**: 三个编辑器 (Script 节点 / Expr 表达式 / yt 控制台) 共享 `frontend/src/components/expressions/CodeEditor.vue` 主体 + per-mode 配置 (extensions builder / 补全源 / 参考文档 / commentable·foldable·formattable); 放大壳 = `EditorModal`, 控制台 = `YtConsoleModal`。编辑器 modal `dismissible=false`(补全浮层挂 body 防点外部误关)。撤销引擎 `historyEngine`(纯, 历史条目带 sgState → 主图+子图统一撤销)。细节 [docs/yt-scripting-console](docs/yt-scripting-console.md)。
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **42** = misc-tools-backlog 未翻译 UI(含本分支 chrome/launcher 新增: SettingsLauncher/FloatingLauncherView/HudShell/IconPicker + 1 处 console.log) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 **18** 错(oxlint 1.64 新规则; 2026-06-12 实测修正, 旧账 16 系漂移, 全部已实证存在于 HEAD 同款代码)。跑全套测试/检查时按此判红。
