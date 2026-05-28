// useEditorSave onSave + 孤儿 GC 一体化。
// onSaveAndClose 留在 view（依赖 view-local close 状态）。
import type { Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend, type Container } from '@/lib/backend'
import { useContainerEditorStore } from '@/stores/containerEditor'

export function useEditorSave(opts: {
  draft: Ref<Container | null>
  dirty: Ref<boolean>
  gcOrphanSubgraphs: () => string[]
  toast: { add: (o: Record<string, unknown>) => unknown }
}) {
  const { draft, dirty, gcOrphanSubgraphs, toast } = opts
  const editorStore = useContainerEditorStore()
  const { t } = useI18n()

  async function onSave(): Promise<boolean> {
    if (!draft.value) return false
    const patch = JSON.parse(JSON.stringify(draft.value))
    delete patch.subgraphs

    // 孤儿子图 GC
    const orphanIDs = gcOrphanSubgraphs()
    for (const sgID of orphanIDs) {
      try {
        await backend.containers.deleteSubgraph(draft.value.id, sgID)
        console.log('[GC] orphan subgraph deleted', sgID)
      } catch (e) {
        console.warn('[GC] orphan delete failed', sgID, e)
      }
    }

    const ok = await backend.containers.update(draft.value.id, JSON.stringify(patch))
    if (ok === undefined) {
      toast.add({ title: t('editorSave.main_save_failed'), color: 'error' })
      return false
    }
    const orphanSet = new Set(orphanIDs)
    const failed: string[] = []
    for (const sg of editorStore.subgraphsForCurrentContainer) {
      if (orphanSet.has(sg.id)) continue
      const sgPatch = JSON.stringify(sg)
      try {
        await backend.containers.updateSubgraph(draft.value.id, sg.id, sgPatch)
      } catch (e) {
        console.warn('updateSubgraph failed', sg.id, e)
        failed.push(sg.id)
      }
    }
    if (failed.length > 0) {
      toast.add({
        title: t('editorSave.subgraph_save_failed', { n: failed.length }),
        description: failed.join(', '),
        color: 'error',
      })
      // 保留 dirty=true，用户可重试 / 不被误导认为已保存
      return false
    }
    toast.add({ title: t('toast.saved'), color: 'success', icon: 'i-tabler-check' })
    dirty.value = false
    return true
  }

  return { onSave }
}
