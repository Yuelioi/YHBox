import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ui from '@nuxt/ui/vue-plugin'
import { useToast } from '@nuxt/ui/composables'
import { useDark } from '@vueuse/core'

// Node registry: 启动期 RPC 拉 backend Spec → adapter 转 → 注册到老 byKind.
// 替代手写 specs/*.ts. 各 consumer (ContextMenu / NodeExplorerModal / etc) 不动.
// populateRegistryFromBackend 在 boot async 内 mount 前 await (见底).
import { populateRegistryFromBackend } from '@/components/containers/nodeRegistry/adapter'

import App from './App.vue'
import { router } from './router'
import { wireEvents } from './lib/events'
import { setupInvoker } from './lib/invoke'
import { useSettingsStore } from './stores/settings'
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
  // 节点 registry RPC populate: 必须 mount 前, 否则 ContextMenu / NodeExplorerModal 等
  // consumer 用 allSpecs() 拿空. backend NodeService.GetAllNodeSpecs 是 SoT.
  await populateRegistryFromBackend()
  app.mount('#app')
})()
