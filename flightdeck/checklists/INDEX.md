# checklists/ — INDEX

<!-- AUTO:checklists -->
- [2026-06-05-node-data-flow.md](2026-06-05-node-data-flow.md) — active — when_to_read: 设计让节点消费/产出数据的节点前; 连线节点间数据前; 撞 INVALID_PIN 'out pin X 不存在' 或数据连不上时 — applies_to: [node-data-flow, data-edge, exec-edge, capture, output-capture, GetVar, Now, VarLastChange, pin-wiring, validator, node-design]
- [2026-06-06-settings-page-style.md](2026-06-06-settings-page-style.md) — active — when_to_read: 写/改任何设置 tab（SettingsView 下的子页：General/Hotkeys/Input/Launcher 等）前；新增设置子页前；想让设置页风格统一时 — applies_to: [frontend, vue, settings, style, consistency, nuxtui, SettingsView]
- [add-node.md](add-node.md) — active — when_to_read: 新增 / 改一个节点 kind 前 (backend Spec → 前端渲染 → 面板 → i18n 全链路) — applies_to: [node, add-node, nodepkg, spec, palette, i18n, registry, frontend, backend]
- [build.md](build.md) — active — when_to_read: before compiling / building / verifying production artifact / 跑 runtime 测试套件 / 真机 smoke — applies_to: [build, compile, task-dev, task-build, wails, exe, vite, bindings, smoke, test-fixture]
- [code-style.md](code-style.md) — active — when_to_read: before writing / editing / deleting source code (.go / .ts / .vue) — including comments / 删疑似死代码或符号前 — applies_to: [code-style, comments, naming, go, typescript, vue, dead-code, delete-symbol, grep]
- [comments.md](comments.md) — active — when_to_read: before writing or editing any source-code comment — applies_to: [comments, code-style, documentation]
- [commits.md](commits.md) — active — when_to_read: before writing a commit message / staging files / preparing a PR — applies_to: [commit, git, staging, message, push, pr]
- [headless-ui-verify.md](headless-ui-verify.md) — active — when_to_read: 改了前端视图/样式想"亲眼"看渲染对不对又不想开完整 app 时; 用 Playwright MCP 离屏渲染 wails 前端做视觉自检; 给独立窗口/设置页调样式后自检前 — applies_to: [frontend, wails, playwright, mcp, visual-verify, vite, pinia, self-check]
- [node-spec-style.md](node-spec-style.md) — active — when_to_read: before writing / editing 任何 nodepkg.Spec (`internal/nodes/**/*.go`) — 含 Inputs/Outputs/Default — applies_to: [node-spec, pin-naming, convention, go]
- [standalone-window-style.md](standalone-window-style.md) — active — when_to_read: 新建 / 改任何独立工具窗 (frameless HUD —— 录屏 / 截图 / 鼠标检测 / 校准 / 悬浮窗启动器 等) 前; 想让这些窗口风格统一 / "像一个产品" 时 — applies_to: [frontend, wails, hud, standalone-window, style, HudShell, consistency, tools-service]
- [ui.md](ui.md) — active — when_to_read: before writing / editing .vue components or Tailwind classes — applies_to: [vue, tailwind, nuxtui, ui, frontend, component, modal, button]
- [vue-i18n-message-compiler-traps.md](vue-i18n-message-compiler-traps.md) — active — when_to_read: 写 / 改 zh.ts / en.ts 翻译 (尤其含 `{` `}` `|` `@` `$` / JSON literal / 管道符 / 邮箱样例); 派 subagent 批量写 i18n 前; 改 vite.config 的 unplugin-vue-i18n 配置; UI 显示 `{n}` 字面 placeholder 没替换 / 弹 SyntaxError 整个组件挂掉 — applies_to: [frontend, vue, i18n, vue-i18n, vite, unplugin-vue-i18n, message-compiler, subagent-dispatch]
<!-- /AUTO -->
