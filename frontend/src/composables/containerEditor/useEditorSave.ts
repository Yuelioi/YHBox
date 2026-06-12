// useEditorSave onSave。onSaveAndClose 留在 view（依赖 view-local close 状态）。
// 子图全局化 (2026-06-12): 只保存本容器编辑会话动过的子图 (touchedFor — 归属清晰,
// 不跨容器代保), 带基准 rev 乐观锁; 被拒 (盘上已有更新) → 走 staleSubgraphs 提示重载。
import { ref, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend, type Container } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'
import { useContainerEditorStore } from '@/stores/containerEditor'

export function useEditorSave(opts: {
  containerID: string
  draft: Ref<Container | null>
  dirty: Ref<boolean>
  toast: { add: (o: Record<string, unknown>) => unknown }
}) {
  const { containerID, draft, dirty, toast } = opts
  const editorStore = useContainerEditorStore()
  const { t } = useI18n()

  // 保存成功的内联反馈窗口 (工具栏按钮闪「已保存」) — 成功不弹 toast, 反馈在触发点。
  const saveFlash = ref(false)
  let flashTimer = 0

  // 乐观锁被拒的子图 (盘上 rev 更新 = 别的编辑器已保存) — view 据此弹"重载?"对话框。
  const staleSubgraphs = ref<string[]>([])

  async function onSave(): Promise<boolean> {
    if (!draft.value) return false

    // 子图必须先于主图落盘: 主图保存带全量校验, 引用的子图还没在盘上会 MISSING_SUBGRAPH 误炸。
    const failed: string[] = []
    const stale: string[] = []
    const savedIDs: string[] = []
    for (const id of editorStore.touchedFor(containerID)) {
      const sg = editorStore.subgraphById(id)
      if (!sg) continue // 已被删 (级联删除等) — 无需保存
      try {
        await backend.subgraphs.updateSilent(id, JSON.stringify(sg), sg.rev ?? 0)
        savedIDs.push(id)
        // 回灌新 rev: 保存成功后盘上 rev+1, 不刷新会让下次保存被自己拒掉。
        const fresh = await backend.subgraphs.get(id)
        if (fresh) editorStore.replaceSubgraph(fresh)
      } catch (e) {
        // 后端 ErrSubgraphRevStale 的稳定可机检段是英文前缀 "subgraph rev stale"。
        const msg = errorMessage(e)
        if (msg.includes('rev stale')) {
          stale.push(id)
        } else {
          console.warn('subgraph update failed', id, e)
          failed.push(id)
        }
      }
    }
    editorStore.clearTouched(containerID, savedIDs)

    if (stale.length > 0) {
      // 乐观锁拒绝 — 不 toast 刷屏, 交给 view 弹"盘上已有更新, 重载?"对话框。
      staleSubgraphs.value = stale
      return false
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

  // 用户在"盘上已有更新"对话框选了重载 → 放弃本地这些子图的修改, 取盘上版本。
  async function reloadStaleSubgraphs(): Promise<void> {
    const ids = staleSubgraphs.value
    staleSubgraphs.value = []
    for (const id of ids) {
      const fresh = await backend.subgraphs.get(id)
      if (fresh) editorStore.replaceSubgraph(fresh)
    }
    editorStore.clearTouched(containerID, ids)
  }
  function dismissStaleSubgraphs() {
    staleSubgraphs.value = []
  }

  return { onSave, saveFlash, staleSubgraphs, reloadStaleSubgraphs, dismissStaleSubgraphs }
}
