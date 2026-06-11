/// <reference types="vitest/config" />
import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import ui from "@nuxt/ui/vite";
import wails from "@wailsio/runtime/plugins/vite";
import VueI18nPlugin from "@intlify/unplugin-vue-i18n/vite";
import { fileURLToPath, URL } from "node:url";

export default defineConfig({
  test: {
    environment: "happy-dom",
    globals: true,
    include: ["src/**/*.{test,spec}.{ts,tsx}"],
  },
  // 终端错误别被 vite 启动 banner 清屏冲掉——dev 跟 wails 串行起，
  // 看不到 vite 报错就以为是 wails 卡住。
  clearScreen: false,
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
    // HMR 显式走同端口的 ws。Windows webview2 在某些设备上对 host autodetect 不稳，
    // 这里钉死 127.0.0.1 + ws，避免 hmr client 连到 0.0.0.0 / IPv6 失败。
    hmr: {
      host: "127.0.0.1",
      port: Number(process.env.WAILS_VITE_PORT) || 9245,
      protocol: "ws",
    },
    // Windows fsnotify 偶尔漏文件（尤其编辑器跨盘符 / OneDrive），开 polling 保底；
    // 代价是 CPU 多 1-2%，对 dev 体验远比丢更改友好。
    watch: {
      usePolling: true,
      interval: 200,
    },
    // dev 模式禁止任何静态资源缓存——webview2 默认 cache 比较激进，
    // 改前端文件后哪怕 vite 重编了，webview 也可能用缓存的旧 module，HMR 看着像没生效。
    headers: {
      "Cache-Control": "no-store, no-cache, must-revalidate",
    },
  },
  plugins: [
    vue(),
    ui({
      ui: {
        colors: {
          primary: "emerald",
          neutral: "zinc",
          // 项目警告色一直是 amber 系 (NuxtUI 默认 yellow) — 钉死防漂移
          warning: "amber",
        },
      },
    }),
    VueI18nPlugin({
      include: [fileURLToPath(new URL("./src/i18n/*.yaml", import.meta.url))],
      // jitCompilation:false 让 plugin 在 build 时把 yaml messages 预编译成函数；
      // 不开这个的话默认 jit 模式会在浏览器 runtime 调 message compiler，
      // 某些 unicode 字符（如全角括号）会让它 throw SyntaxError 让整个 view 挂掉。
      jitCompilation: false,
      // 必须显式 false: 插件默认 runtimeOnly=true 会把 `vue-i18n` alias 到 runtime-only
      // bundle (无 message compiler), 导致 `t('x', { n: 5 })` 这类带 placeholder 的
      // 翻译字符串无人 compile, 字面渲染 `{n}` 出来. zh.ts 全用 plain string + ts
      // 模式喂 createI18n, 必须保留 compiler.
      runtimeOnly: false,
    }),
    wails("./bindings"),
  ],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
      "@bindings": fileURLToPath(new URL("./bindings", import.meta.url)),
    },
  },
});
