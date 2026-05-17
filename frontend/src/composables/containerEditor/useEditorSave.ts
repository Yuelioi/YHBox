// useEditorSave onSave + 孤儿 GC 一体化。
// onSaveAndClose 留在 view（依赖 view-local close 状态）。
import type { Ref } from 'vue'
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

  async function onSave(): Promise<boolean> {
    if (!draft.value) return false
    const patch = JSON.parse(JSON.stringify(draft.value))
    delete patch.subgraphs

    // 孤儿子图 GC
    const orphanIDs = gcOrphanSubgraphs()
    for (const sgID of orphanIDs) {
      try {
        await backend.containers.deleteSubgraph(draft.value.id, sgID)
        console.log('[GC] 删孤儿子图', sgID)
      } catch (e) {
        console.warn('[GC] 删孤儿失败', sgID, e)
      }
    }

    const ok = await backend.containers.update(draft.value.id, JSON.stringify(patch))
    if (ok === undefined) {
      toast.add({ title: '主图保存失败', color: 'error' })
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
        title: `${failed.length} 个子图保存失败`,
        description: failed.join(', '),
        color: 'error',
      })
      // 保留 dirty=true，用户可重试 / 不被误导认为已保存
      return false
    }
    toast.add({ title: '已保存', color: 'success', icon: 'i-tabler-check' })
    dirty.value = false
    return true
  }

  return { onSave }
}
