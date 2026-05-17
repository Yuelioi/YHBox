// stores/library.ts 库 store — 缓存 subgraphs + templates，订阅后端 library:changed 事件刷新。
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { backend, type LibrarySubgraph, type LibraryTemplate } from '@/lib/backend'

export const useLibraryStore = defineStore('library', () => {
  const subgraphs = ref<LibrarySubgraph[]>([])
  const templates = ref<LibraryTemplate[]>([])
  const loading = ref(false)

  async function reload() {
    loading.value = true
    try {
      subgraphs.value = ((await backend.library.listSubgraphs()) as LibrarySubgraph[]) ?? []
      templates.value = ((await backend.library.listTemplates()) as LibraryTemplate[]) ?? []
    } catch (e) {
      console.warn('library.reload failed', e)
    } finally {
      loading.value = false
    }
  }

  // 后端 library:changed 事件订阅（跨窗口同步）
  Events.On('library:changed', () => {
    void reload()
  })

  return { subgraphs, templates, loading, reload }
})
