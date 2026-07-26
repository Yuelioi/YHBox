import { defineStore } from 'pinia'
import { ref } from 'vue'
import { backend, type HotkeyEntry } from '@/lib/backend'

// 所有热键的中央 store。SettingsHotkeys.vue 用 list 渲染表格。
// hotkey:changed 事件订阅在 events.ts，触发 reload。
export const useHotkeysStore = defineStore('hotkeys', () => {
  const list = ref<HotkeyEntry[]>([])

  async function reload() {
    list.value = (await backend.hotkeys.list()) as unknown as HotkeyEntry[]
  }

  async function update(key: string, hotkeyStr: string) {
    await backend.hotkeys.update(key, hotkeyStr)
    await reload()
  }

  // keyFor 按 registry key 取当前绑定的热键串 (响应式, 跟 list 走). 未找到/未绑定回退 fallback.
  // 用于把"按 F12 停止"这类硬编显示换成实时绑定值 (rebind 后自动更新)。
  function keyFor(registryKey: string, fallback: string): string {
    return list.value.find((e) => e.key === registryKey)?.hotkeyStr || fallback
  }

  return { list, reload, update, keyFor }
})
