# 独立工具窗统一样式 checklist

SUMMARY: frameless 独立工具窗共用 HudShell + 设计 token 的统一样式约定, 让所有 HUD 像一个产品
READ WHEN: 新建 / 改任何独立工具窗 (frameless HUD —— 录屏 / 截图 / 鼠标检测 / 校准 / 悬浮窗启动器 等) 前; 想让这些窗口风格统一 / "像一个产品" 时
RECHECK WHEN: 改 HudShell / frameless 独立工具窗的公共风格约定时

---

项目有一组 frameless + AlwaysOnTop 的独立窗口 (录屏 HUD / 截图选择器 / 鼠标检测 HUD / 校准 HUD / 悬浮窗启动器)。它们**必须看起来像一个产品** —— 共用一套 chrome，别各写各的。

## 共享外壳 `components/tools/HudShell.vue`

所有独立窗口的 root + 标题栏走 `HudShell`，**不要重复手写**：

- props: `title` / `icon` / `accent`(primary|error|success|warning|neutral) / `dense`(紧凑标题栏，工具条用) / `noClose` / `closeTitle`
- slots: 默认 = body；`#actions` = 标题栏右侧额外按钮 (如启动器的图钉)
- emit: `close` (右侧 × 按钮触发，各窗口自己接 —— 有的 `Window.Close()`，有的走后端 Hide)
- 标题栏自带 `--wails-draggable: drag`；actions 区 `no-drag`

样例：`FloatingLauncherView.vue` (dense + 图钉 action + @close→hideLauncher)。

## 后端开窗 (`internal/services/tools/service.go`)

仿 `OpenRecordingHUD` (该文件已有 4 个范例)：

- `app.Window.NewWithOptions({ Frameless:true, AlwaysOnTop:true, BackgroundColour: application.NewRGB(18,18,18), URL:"/#/tools/xxx", ... })`
- Service 存窗口句柄字段；**已开则 `Focus()` 防重复开**；`OnWindowEvent(events.Common.WindowClosing)` 清引用
- 运行时改尺寸/置顶：wails3 `WebviewWindow` 有 `SetSize(w,h)` / `SetAlwaysOnTop(b)` / `Show()` / `Hide()` (webview_window.go)

## 路由

`router/index.ts` 加 `meta: { standalone: true }` —— `App.vue` 据此跳过主应用壳 (sidebar/titlebar)，只渲染裸 view。

## 设计 token (跟主题，别硬编码颜色)

- 根底 `bg-default`；边框 `border-default`；卡片/凸起面 `bg-elevated/40`
- 文字 `text-highlighted`(主) / `text-toned`(次) / `text-dimmed`(弱)；强调用 `text-primary`/`text-error` 等
- 数字用 `font-mono tabular-nums`；按钮统一 `UButton size="xs"`
- 过渡 150–300ms (`transition-colors` 等)；minimal chrome，别堆装饰

## 状态面板 (HUD body 内容区)

HUD 各状态 (校准: 等待/倒计时/录制/完成；录制: 倒计时/录制/暂停/继续/待机) 用**彩色状态面板**传达语义，**别只堆居中纯文字**：

- 每态一个 `w-full rounded-lg border border-{色}/40 bg-{色}/10 px-4 py-3 text-center space-y-1`
- 色按语义：进行/倒计时 `primary` 或 `amber-500`、录制/危险 `error`、成功/完成 `success`/`primary`、等待中性 `border-dashed border-default/60 bg-elevated/40`
- 范例：`CalibrationHUDView`(校准三档) / `RecordingHUDView`(录制五态) —— 两窗共用这套观感。

## 两个坑

- **置顶盖不住独占全屏**：AlwaysOnTop 在独占全屏游戏上盖不住 (Windows 层限制)，窗口化/无边框全屏 OK。文档化即可，别想在这解决。
- **跨窗口状态不同步**：独立窗口是另一个 webview = **另一个 pinia store**。主程序改了 settings/数据，独立窗口收不到。要后端 `app.Emit("xxx:changed")` + 独立窗口 `Events.On(...)` reload，否则"改了没反应"。(悬浮窗 icon 不生效就是栽在这。)

## 视觉自检

改完别只 typecheck 就交 —— 照 [headless-ui-verify](headless-ui-verify.md) 用 vite + Playwright 离屏渲染**亲眼看**，或让用户真机 smoke。"我觉得对"屡次翻车。
