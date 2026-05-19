import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ui from '@nuxt/ui/vue-plugin'
import { useToast } from '@nuxt/ui/composables'
import { useDark } from '@vueuse/core'

// Side-effect: register all v4 node kinds in nodeRegistry. MUST be the first
// import here — other modules (pinSpec.ts / nodeFieldSchemas.ts / NodePalette.vue
// / stores) derive top-level const maps from the registry at module-init time,
// so the registry must be populated before any of them evaluates. Other shells
// (pinSpec.ts, nodeFieldSchemas.ts) used to duplicate this side-effect import;
// centralized here as single source of truth.
import '@/components/containers/nodeRegistry/specs'

import App from './App.vue'
import { router } from './router'
import { wireEvents } from './lib/events'
import { setupInvoker } from './lib/invoke'
import { useSettingsStore } from './stores/settings'
import { useGameStore } from './stores/game'
import { i18n } from './i18n'

import './style.css'

const app = createApp(App)
const pinia = createPinia()
app.use(pinia).use(router).use(ui).use(i18n)

// 强制 dark：NuxtUI 的 vue-plugin install() 调 @vueuse/core useDark()，
// 它会读 localStorage 和系统偏好动态写 <html class>，把 index.html 静态写的
// class="dark" 在挂载时覆盖。这里调一次 useDark() 拿到 ref 设回 true，
// 永久锁 dark（localStorage 里也会持久化）。app 永远不切 light。
useDark().value = true

// Wire invoke toast inside Vue injection context (before mount)
app.runWithContext(() => {
  const toast = useToast()
  setupInvoker((opts) => toast.add(opts))
})

// Subscribe Go → JS events
wireEvents()

// Hydrate then mount
;(async () => {
  await useSettingsStore().load()
  useGameStore().detect()
  app.mount('#app')
})()
