# Frontend UI development

## Current sources of truth

- Nuxt UI theme/colors/component defaults：`frontend/vite.config.ts`。
- dark surface、semantic utilities 和全局 component baseline：`frontend/src/style.css`。
- 通用 modal：`frontend/src/components/common/BaseModal.vue`。
- settings hierarchy：`frontend/src/components/settings/`。
- Wails transport：`frontend/src/lib/backend.ts` 和 `frontend/src/app/transport/workflow.ts`。
- i18n：`frontend/src/i18n/zh.ts`、`en.ts`；组件不硬编码只面向一种语言的布局。

## Visual contract

- Yotta 固定 dark mode；`index.html`/app root 提供 dark class，`main.ts` 用 `useDark().value = true` 防止
  Nuxt UI 根据系统偏好覆盖。
- 普通 chrome 使用 Nuxt UI semantic token：`bg-default/elevated/muted/accented`、
  `text-highlighted/default/toned/muted/dimmed`、`border-*` 和 `primary/error/warning/success/info`。
- `bg-sunken` 只用于终端、轨道等凹陷面；`raised-surface`、`overlay-surface` 和 `workspace-*` 的定义都在
  `style.css`。页面不要散写 zinc/black/white 来创建自己的暗色系统。
- 有 Nuxt UI 原子组件时使用它；自研复合组件仍由 `UButton`、`UInput`、`UModal`、`UTooltip` 等组合。
  真正需要原生 drag/canvas/semantic element 时可以用裸 HTML，但视觉仍走 semantic token。
- 普通 modal 使用 `BaseModal`；特殊 command palette、重型 asset browser 或独立 frameless window 可以有
  自己的结构，但不能复制一套全局 modal/颜色规范。
- 设置页使用 `SettingsPageHeader`、`SettingsSection`、`SettingsRow` 等共享层级；section 内字段不重复套
  完整 card。短数值/端口控件显式限制宽度，长文本/普通表单默认占满可用宽度。
- 一个任务区只有一个明确主操作；选择态、错误态和禁用态除颜色外还要有文字、图标、边框或形状线索。

## Feedback and errors

错误 ID、params、transport fallback 和持久证据的完整规则见[错误契约](../errors/error-contract.md)。

一个事件只拥有一个主要反馈表面：

- 字段错误放字段旁，保留输入；当前区域可恢复错误使用 inline alert。
- 操作结果在当前视野已经可见时不额外 toast；复制/保存优先在触发按钮原地短暂确认。
- 后台、跨窗口或触发点消失的结果才使用 toast；首次加载无法继续使用 page failure state。
- 只有必须阻止继续且需要用户选择时才用 modal。
- 恢复建议只能指向当前产品中真实存在、用户可找到的入口；不存在诊断页面时不能写“打开诊断信息”。
- RPC transport 只 normalize/rethrow typed error，不自动 toast；domain action 决定最终反馈表面，不能吞错后
  以 `undefined`/`false` 冒充成功。

## Verification

1. 运行与改动相邻的 component/domain Vitest；响应式状态要断言用户可见结果，不只断言内部 ref。
2. 运行前端 format/lint/typecheck/i18n gate（通常由 `task check` 路由）。
3. 用 CLI Playwright 启动本地页面，覆盖真实 viewport、keyboard/mouse 交互、focus 和错误/空/加载态，并
   实际查看 PNG。不要把 DOM 存在当作视觉通过，也不要依赖内置浏览器。
4. Wails binding、event、window 或 WebView 行为使用 `task webview:smoke`；纯 Vite 页面不能证明 native RPC、
   window sizing、global hook 或 platform adapter。
5. screenshot 断言要允许 Chromium 返回 CSS Color 4 表达；比较 semantic surface 时解析 CSS variable，
   最终仍以真实截图检查布局、层级、溢出和可读性。
