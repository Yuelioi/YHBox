import { defineStore } from 'pinia'
import { ref } from 'vue'
import { backend, type HotkeyEntry } from '@/lib/backend'

// 所有热键的中央 store。SettingsHotkeys.vue 用 list 渲染表格。
// hotkey:changed 事件订阅在 events.ts，触发 reload。
export const useHotkeysStore = defineStore('hotkeys', () => {
  const list = ref<HotkeyEntry[]>([])

  async function reload() {
    const r = await backend.hotkeys.list()
    if (r !== undefined) list.value = r as unknown as HotkeyEntry[]
  }

  async function update(key: string, hotkeyStr: string) {
    const r = await backend.hotkeys.update(key, hotkeyStr)
    if (r === undefined) return false
    await reload()
    return true
  }

  return { list, reload, update }
})
