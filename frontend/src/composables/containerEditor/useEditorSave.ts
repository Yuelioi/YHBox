// useEditorSave onSave。onSaveAndClose 留在 view（依赖 view-local close 状态）。
// 不在保存路径做孤儿子图 GC —— 删子图只在用户显式删 Subgraph 节点时由 deleteSubgraphCascade 联动，
// 否则自动保存撞上「store 快照落后于编辑真相」会误删仍被引用的子图 (录制雪崩根因)。
import { ref, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend, type Container } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'
import { useContainerEditorStore } from '@/stores/containerEditor'

export function useEditorSave(opts: {
  draft: Ref<Container | null>
  dirty: Ref<boolean>
  toast: { add: (o: Record<string, unknown>) => unknown }
}) {
  const { draft, dirty, toast } = opts
  const editorStore = useContainerEditorStore()
  const { t } = useI18n()

  // 保存成功的内联反馈窗口 (工具栏按钮闪「已保存」) — 成功不弹 toast, 反馈在触发点。
  const saveFlash = ref(false)
  let flashTimer = 0

  async function onSave(): Promise<boolean> {
    if (!draft.value) return false

    // 子图必须先于主图落盘: 主图保存带全量校验, 引用的子图还没在盘上会 MISSING_SUBGRAPH 误炸。
    const failed: string[] = []
    for (const sg of editorStore.subgraphsForCurrentContainer) {
      try {
        await backend.containers.updateSubgraphSilent(draft.value.id, sg.id, JSON.stringify(sg))
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
      // 子图没存全就别碰主图: 主图引用可能指向没存上的子图。保留 dirty=true 让用户重试。
      return false
    }

    const patch = JSON.parse(JSON.stringify(draft.value))
    delete patch.subgraphs
    try {
      await backend.containers.updateSilent(draft.value.id, JSON.stringify(patch))
    } catch (e) {
      // 一条 toast 收口: 标题「主图保存失败」+ 本地化原因, 不再叠 invoke 的自动 toast。
      toast.add({ title: t('editorSave.main_save_failed'), description: errorMessage(e), color: 'error' })
      return false
    }

    dirty.value = false
    saveFlash.value = true
    window.clearTimeout(flashTimer)
    flashTimer = window.setTimeout(() => { saveFlash.value = false }, 1600)
    return true
  }

  return { onSave, saveFlash }
}
