import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { Events } from '@wailsio/runtime'
import { backend, type SubgraphPackage, type Subgraph, type ImportResult, type VarDecl } from '@/lib/backend'
import { toastInfo } from '@/lib/invoke'
import { i18n } from '@/i18n'

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

  // 导入子图到容器, 并自动补齐目标容器缺的 required globals (子图要用的变量本就必须有).
  // 用于容器编辑器里的直接导入 (拖拽/点选) — 这些路径不走带提示步的 ImportToContainerDialog.
  // 补了变量则 toast 告知. 返回导入项 + 实际补的变量名.
  async function importEnsuringGlobals(
    libSgID: string,
    containerID: string,
  ): Promise<{ imported: { kind: string; key: string }[]; added: string[] } | undefined> {
    const result = await backend.library.importToContainer(libSgID, containerID)
    if (!result) return undefined
    const r = result as ImportResult
    const missing = r.missingGlobals ?? []
    let added: string[] = []
    if (missing.length > 0) {
      const c = await backend.containers.get(containerID)
      if (c) {
        const existing = (c.vars ?? []) as VarDecl[]
        const names = new Set(existing.map((v) => v.name))
        const toAdd: VarDecl[] = missing
          .filter((g) => !names.has(g.name))
          .map((g) => ({
            name: g.name,
            type: (g.type || 'any') as VarDecl['type'],
            default: (g.default ?? null) as VarDecl['default'],
          }))
        if (toAdd.length > 0) {
          await backend.containers.update(containerID, JSON.stringify({ vars: [...existing, ...toAdd] }))
          added = toAdd.map((v) => v.name)
        }
      }
    }
    if (added.length > 0) {
      toastInfo(i18n.global.t('library.import.auto_added_vars', { n: added.length, names: added.join(', ') }))
    }
    return { imported: r.imported ?? [], added }
  }

  Events.On('library:changed', () => {
    void reload()
  })

  return { subgraphIds, packages, subgraphs, loading, reload, deletePackage, importEnsuringGlobals }
})
