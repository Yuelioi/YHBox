---
kind: checklist
summary: "frameless 独立工具窗共用 HudShell + 设计 token 的统一样式约定, 让所有 HUD 像一个产品"
activation: action
read_when: "新建 / 改任何独立工具窗 (frameless HUD —— 录屏 / 截图 / 鼠标检测 / 校准 / 悬浮窗启动器 等) 前; 想让这些窗口风格统一 / \"像一个产品\" 时"
recheck_when: "改 HudShell / frameless 独立工具窗的公共风格约定时"
---
# 独立工具窗统一样式 checklist
项目有一组 frameless + AlwaysOnTop 的独立窗口 (录屏 HUD / 截图选择器 / 鼠标检测 HUD / 校准 HUD / 悬浮窗启动器)。它们**必须看起来像一个产品** —— 共用一套 chrome，别各写各的。

## 共享外壳 `components/tools/HudShell.vue`

所有独立窗口的 root + 标题栏走 `HudShell`，**不要重复手写**：

- props: `title` / `subtitle` / `icon` / `accent`(primary|error|success|warning|neutral) / `status` / `statusActive` / `dense`(紧凑标题栏，工具条用) / `noClose` / `closeTitle`
- slots: 默认 = body；`#actions` = 标题栏右侧额外按钮 (如启动器的图钉)
- emit: `close` (右侧 × 按钮触发，各窗口自己接 —— 有的 `Window.Close()`，有的走后端 Hide)
- 标题栏自带 `--wails-draggable: drag`；actions 区 `no-drag`
- 标题栏、状态胶囊和关闭按钮已经处理窄窗收缩、`aria-label` 与容器查询；窗口内不要再套第二层自制标题栏

样例：`FloatingLauncherView.vue` (dense + 图钉 action + @close→hideLauncher)。

## 后端开窗 (`wails_tools.go`)

`wailsToolsWindowOptions` 是全部工具窗的 presentation policy：

- 在这里统一维护初始宽高、最小宽高、`Frameless` / `AlwaysOnTop` / `DisableResize` 和路由，不要把尺寸散落到 service
- 调整尺寸时同步改 `wails_tools_test.go`；默认尺寸必须能完整容纳标题、副标题、状态面板与底部操作
- 可缩放窗口必须给真实可用的最小尺寸；截图选择器当前最小 `760×520`，小 HUD 不低于 `300×240`，启动器不低于 `200×96`
- 运行时改尺寸/置顶：wails3 `WebviewWindow` 有 `SetSize(w,h)` / `SetAlwaysOnTop(b)` / `Show()` / `Hide()` (webview_window.go)

## 路由

`router/index.ts` 加 `meta: { standalone: true }` —— `App.vue` 据此跳过主应用壳 (sidebar/titlebar)，只渲染裸 view。

## 设计 token (跟主题，别硬编码颜色)

- 根底 `bg-default`；边框 `border-default`；卡片/凸起面 `bg-elevated/40`
- 文字 `text-highlighted`(主) / `text-toned`(次) / `text-dimmed`(弱)；强调用 `text-primary`/`text-error` 等
- 数字用 `font-mono tabular-nums`；图标操作用 `UButton size="xs"`，主要动作可用 `size="sm"`，禁止裸 `<button>` 或可点击 `UIcon`
- 过渡 150–300ms (`transition-colors` 等)；minimal chrome，别堆装饰

## 状态面板 (HUD body 内容区)

HUD 各状态 (校准: 等待/倒计时/录制/完成；录制: 倒计时/录制/暂停/继续/待机) 用共享 `components/tools/HudStatePanel.vue` 传达语义，**别只堆居中纯文字或复制一套 class**：

- `tone` 只用 `primary | error | success | warning | neutral`，再传 `icon` / `eyebrow` / `value` / `hint` / `active` / `size`
- 色按语义：进行/倒计时 `primary` / `warning`、录制/危险 `error`、成功/完成 `success` / `primary`、等待 `neutral`
- `active` 只用于正在发生的实时状态，自动带有遵守 `prefers-reduced-motion` 的状态脉冲
- 范例：`CalibrationHUDView`(校准三档) / `RecordingHUDView`(录制五态) —— 两窗共用这套观感。

## 响应式与内容密度

- 小 HUD 首先保留标题、状态与主动作；`HudShell` 在容器宽度 `≤340px` 隐藏副标题，业务 view 不要用固定横向空白撑宽
- 截图选择器保持“画布优先”：工具栏在上，属性侧栏 `clamp(272px, 24vw, 328px)`；`≤860px` 收到 270px，低高度减少面板 padding
- 需要滚动的内容区必须有 `min-h-0`；鼠标 HUD 与启动器允许 body 内滚动，固定录制/校准 HUD 的默认高度必须无裁切
- 截图、坐标、颜色和快捷键等技术值用等宽字体；说明文本用产品语言并纳入 zh/en i18n，独立窗里不留中文硬编码

## 两个坑

- **置顶盖不住独占全屏**：AlwaysOnTop 在独占全屏游戏上盖不住 (Windows 层限制)，窗口化/无边框全屏 OK。文档化即可，别想在这解决。
- **跨窗口状态不同步**：独立窗口是另一个 webview = **另一个 pinia store**。主程序改了 settings/数据，独立窗口收不到。要后端 `app.Emit("xxx:changed")` + 独立窗口 `Events.On(...)` reload，否则"改了没反应"。(悬浮窗 icon 不生效就是栽在这。)

## 视觉自检

改完别只 typecheck 就交：

- `StandaloneWindows.spec.ts` 必须覆盖全部独立 view 使用 `HudShell`、没有裸 `<button>`，以及共享拖拽/可访问性契约
- `wails_tools_test.go` 必须覆盖默认与最小尺寸
- 照 [headless-ui-verify](../frontend/headless-ui-verify.md) 用 vite + Playwright 离屏渲染**亲眼看**；浏览器通道不可用时必须明确记录并让用户真机 smoke，不能伪报视觉验证
