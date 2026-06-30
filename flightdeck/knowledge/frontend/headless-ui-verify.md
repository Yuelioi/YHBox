# 离屏渲染前端视图做视觉自检 (vite + Playwright MCP) checklist

SUMMARY: 离屏渲染前端视图做视觉自检 (vite + Playwright MCP), 不开完整 app 也能亲眼核样式.
READ WHEN: 改了前端视图/样式想"亲眼"看渲染对不对又不想开完整 app 时; 用 Playwright MCP 离屏渲染 wails 前端做视觉自检; 给独立窗口/设置页调样式后自检前
RECHECK WHEN: 改离屏视觉自检流程 (Playwright MCP / vite 离屏渲染入口 / pinia mock 方式) 时

---


**为什么**：纯 typecheck/lint 过 ≠ 长得对。本会话多次"我觉得对"翻车 (步进器全宽、布局臃肿)，**截图一看就抓到**。wails app 是桌面 webview 不好直接截，但前端是 vite 服务，可在浏览器里渲染 + 注入 mock 数据看布局/样式。

## 步骤

1. **起 vite** (避开 `task dev` 的 9245)：
   `npm --prefix frontend run dev -- --port 9246 --strictPort` (后台跑)

2. **让 app 在无后端时也能挂载**：`main.ts` 在 `app.mount` 前 `await` 后端 (settings / node registry)，纯 vite 无 wails 会卡死不挂载。**临时**把那两个 await 包 `try { } catch {}` 让它失败也挂载。⚠️ 截完**必须还原**，别提交 (会掩盖真实启动失败)。

3. **找到能渲染的 URL**：
   - `meta.standalone` 的视图 (`/#/tools/xxx`) 直接渲染 (App.vue 跳过壳)。
   - 非 standalone 页 (如 `/#/settings`) 走完整 app 壳 → 无后端会崩成黑屏。**临时**加个 standalone 预览路由指向那个组件 (如 `/_preview/settings-launcher`)，截完还原。

4. **注入 mock 数据** (Playwright `browser_evaluate`)：拿 pinia →
   ```js
   const app = document.querySelector('#app').__vue_app__
   let pinia; for (const s of Object.getOwnPropertySymbols(app._context.provides)) {
     const v = app._context.provides[s]; if (v && v._s instanceof Map) { pinia = v; break } }
   pinia._s.get('settings').data = { ui: { ... } }   // 直接灌 store state
   pinia._s.get('containers').list = [ ... ]
   ```
   (`browser_navigate` 到新 URL 会重载页面 → 注入要在导航后重做。)

5. **截图 + 看**：`browser_take_screenshot` → 用 Read 读 PNG 亲眼核。可 `browser_resize` 到真实窗口尺寸 (如悬浮窗 240×300) 看真实观感。

6. **还原 + 清理**：还原 main.ts / 预览路由；删截图 PNG + `.playwright-mcp/` (别提交)。

## Playwright MCP 不在时的备选 (验证过可用)

会话没接 Playwright MCP → `pnpm add -D puppeteer-core` (零浏览器下载) + 系统 Edge:

```js
import puppeteer from 'puppeteer-core'
const browser = await puppeteer.launch({
  executablePath: 'C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe',
  headless: 'new', defaultViewport: { width: 1680, height: 1000 },
})
```

- mock 数据可以不走 browser_evaluate: 临时预览组件自己往 pinia store 灌假数据更省事。
- 坑1: `pnpm run dev -- --port 9246` 的 `--` 会被原样传给 vite, 端口不生效 — 直接 `pnpm exec vite --port 9246` 或认下 vite.config 的 9245。
- 坑2: modal 打开后 `page.click('.cm-content')` 点到的是**背后小框的编辑器** (DOM 第一个) — 取 `page.$$('.cm-content')` 最后一个。
- 截完: `pnpm remove puppeteer-core` + 删脚本/PNG/临时预览路由, 跟 main.ts 还原一起做。

## 局限

- 后端 RPC (SetSize / 跑容器 / Events) 在纯 vite **看不到效果** —— 只验**布局/样式/组件渲染**。功能行为 (置顶、自适应尺寸、事件刷新) 留**真机 smoke**。
- 控制台会刷一堆 RPC 失败 error，正常，忽略。

配套：独立窗口样式规范见 [standalone-window-style.md](../wails/standalone-window-style.md)。
