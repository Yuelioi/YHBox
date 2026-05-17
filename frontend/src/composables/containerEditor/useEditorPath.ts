// useEditorPath 编辑路径包装：
//   - editorPath（store 转发）
//   - sgLabel(sgID)：从 store 查 label，找不到返回 id
//   - currentSubgraph：editorPath 末尾对应的 subgraph object（null 时 = 主图）
//   - pushPath / popPath / gotoPathIndex（store 转发）
import { computed } from 'vue'
import { useContainerEditorStore } from '@/stores/containerEditor'

export function useEditorPath() {
  const editorStore = useContainerEditorStore()

  const editorPath = computed(() => editorStore.editorPath)

  function sgLabel(sgID: string): string {
    const sg = editorStore.subgraphsForCurrentContainer.find((s) => s.id === sgID)
    return sg?.label ?? sgID
  }

  const currentSubgraph = computed(() => {
    if (editorStore.editorPath.length === 0) return null
    const sgID = editorStore.editorPath[editorStore.editorPath.length - 1]
    return editorStore.subgraphsForCurrentContainer.find((s) => s.id === sgID) ?? null
  })

  return {
    editorPath,
    sgLabel,
    currentSubgraph,
    pushPath: (id: string) => editorStore.pushPath(id),
    popPath: () => editorStore.popPath(),
    gotoPathIndex: (i: number) => editorStore.gotoPathIndex(i),
  }
}
