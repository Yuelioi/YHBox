/// <reference types="vitest/config" />
import { readFile } from 'node:fs/promises'
import { createRequire } from 'node:module'
import { fileURLToPath, URL } from 'node:url'
import { defineConfig, type Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import ui from '@nuxt/ui/vite'
import wails from '@wailsio/runtime/plugins/vite'
import VueI18nPlugin from '@intlify/unplugin-vue-i18n/vite'

const virtualTablerIconNames = 'virtual:tabler-icon-names'
const resolvedVirtualTablerIconNames = `\0${virtualTablerIconNames}`
const require = createRequire(import.meta.url)
const tablerIconsPath = require.resolve('@iconify-json/tabler/icons.json')

function tablerIconNamesPlugin(): Plugin {
  let generatedModule: Promise<string> | undefined

  return {
    name: 'yotta:tabler-icon-names',
    resolveId(id) {
      if (id === virtualTablerIconNames) return resolvedVirtualTablerIconNames
    },
    load(id) {
      if (id !== resolvedVirtualTablerIconNames) return
      generatedModule ??= buildTablerIconNamesModule()
      return generatedModule
    },
  }
}

async function buildTablerIconNamesModule(): Promise<string> {
  const parsed: unknown = JSON.parse(await readFile(tablerIconsPath, 'utf8'))
  if (!isRecord(parsed) || !isRecord(parsed.icons)) {
    throw new Error(`invalid Tabler icon set at ${tablerIconsPath}`)
  }
  const names = Object.keys(parsed.icons).sort()
  if (names.length === 0) throw new Error(`Tabler icon set is empty at ${tablerIconsPath}`)
  return `export default ${JSON.stringify(names)};\n`
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

export default defineConfig(({ mode }) => {
  const isTest = mode === 'test' || process.env.VITEST === 'true'

  return {
    build: {
      // The budget gate resolves logical entry points through this generated manifest.
      manifest: true,
    },
    test: {
      environment: 'happy-dom',
      globals: true,
      include: ['src/**/*.{test,spec}.{ts,tsx}'],
      setupFiles: ['src/test/setup.ts'],
    },
    // 终端错误别被 vite 启动 banner 清屏冲掉——dev 跟 wails 串行起，
    // 看不到 vite 报错就以为是 wails 卡住。
    clearScreen: false,
    server: {
      host: '127.0.0.1',
      port: Number(process.env.WAILS_VITE_PORT) || 9245,
      strictPort: true,
      // HMR 显式走同端口的 ws。Windows webview2 在某些设备上对 host autodetect 不稳，
      // 这里钉死 127.0.0.1 + ws，避免 hmr client 连到 0.0.0.0 / IPv6 失败。
      hmr: {
        host: '127.0.0.1',
        port: Number(process.env.WAILS_VITE_PORT) || 9245,
        protocol: 'ws',
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
        'Cache-Control': 'no-store, no-cache, must-revalidate',
      },
    },
    plugins: [
      tablerIconNamesPlugin(),
      vue(),
      ui({
        ui: {
          colors: {
            primary: 'emerald',
            neutral: 'zinc',
            // 项目警告色一直是 amber 系 (NuxtUI 默认 yellow) — 钉死防漂移
            warning: 'amber',
          },
          // primary + solid 是每个任务区唯一的主操作，不是营销 CTA。
          // 具体 surface/state 由 style.css 的 .btn-primary-contained 从语义 token 派生。
          button: {
            // Nuxt UI base 只有 items-center；固定宽高的 icon-only 按钮会沿主轴贴左。
            // 全局居中，导航/菜单等左对齐按钮用 class="justify-start" 显式覆盖。
            slots: { base: 'justify-center' },
            compoundVariants: [
              {
                color: 'primary',
                variant: 'solid',
                class: { base: 'btn-primary-contained' },
              },
            ],
          },
        },
      }),
      VueI18nPlugin({
        include: [fileURLToPath(new URL('./src/i18n/*.yaml', import.meta.url))],
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
      ...(!isTest ? [wails('./bindings')] : []),
    ],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
        '@bindings': fileURLToPath(new URL('./bindings', import.meta.url)),
        ...(isTest
          ? {
              '@wailsio/runtime': fileURLToPath(
                new URL('./src/test/wailsRuntimeStub.ts', import.meta.url),
              ),
            }
          : {}),
      },
    },
  }
})
