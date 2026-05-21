import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { Events } from '@wailsio/runtime'
import { backend, type SubgraphPackage, type Subgraph } from '@/lib/backend'

export const useLibraryStore = defineStore('library', () => {
  const subgraphIds = ref<string[]>([])
  const packages = ref<Record<string, SubgraphPackage>>({})
  const loading = ref(false)

  // 兼容 NodePalette / LibraryExplorerModal 仍需 subgraphs[] 遍历的旧消费方.
  const subgraphs = computed<Subgraph[]>(() =>
    subgraphIds.value.map((id) => packages.value[id]?.root).filter(Boolean) as Subgraph[],
  )

  async function reload() {
    loading.value = true
    try {
      const ids = ((await backend.library.listSubgraphs()) as string[]) ?? []
      subgraphIds.value = ids
      packages.value = {}
      for (const sgID of ids) {
        const pkg = await backend.library.getSubgraphPackage(sgID)
        if (pkg) packages.value[sgID] = pkg as SubgraphPackage
      }
    } catch (e) {
      console.warn('library.reload failed', e)
    } finally {
      loading.value = false
    }
  }

  async function deletePackage(sgID: string): Promise<boolean> {
    try {
      await backend.library.deleteSubgraphPackage(sgID)
      await reload()
      return true
    } catch (e) {
      console.warn('library.deletePackage failed', e)
      return false
    }
  }

  Events.On('library:changed', () => {
    void reload()
  })

  return { subgraphIds, packages, subgraphs, loading, reload, deletePackage }
})
