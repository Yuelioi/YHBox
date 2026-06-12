import { computed, ref, watch, type Ref } from 'vue'

// 文件管理器式列表多选: 单击替换 / Ctrl toggle / Shift 锚点范围 (按可见顺序)。
// 可见列表变化 (过滤/删除) 自动剔除不可见项。Set 一律整体替换保证响应性。
export function useListSelection(visibleIds: Ref<string[]>) {
  const selected = ref<Set<string>>(new Set())
  const anchorId = ref<string | null>(null)

  function plainSelect(id: string) {
    selected.value = new Set([id])
    anchorId.value = id
  }

  function click(id: string, mods: { ctrl?: boolean; shift?: boolean } = {}) {
    if (mods.shift && anchorId.value !== null) {
      const ids = visibleIds.value
      const a = ids.indexOf(anchorId.value)
      const b = ids.indexOf(id)
      if (a === -1 || b === -1) {
        plainSelect(id)
        return
      }
      selected.value = new Set(ids.slice(Math.min(a, b), Math.max(a, b) + 1))
      return
    }
    if (mods.ctrl) {
      const next = new Set(selected.value)
      if (next.has(id)) {
        next.delete(id)
        if (anchorId.value === id) anchorId.value = null
      } else {
        next.add(id)
        anchorId.value = id
      }
      selected.value = next
      return
    }
    plainSelect(id)
  }

  function clear() {
    selected.value = new Set()
    anchorId.value = null
  }

  watch(visibleIds, (ids) => {
    const vis = new Set(ids)
    if ([...selected.value].some((id) => !vis.has(id))) {
      selected.value = new Set([...selected.value].filter((id) => vis.has(id)))
    }
    if (anchorId.value !== null && !vis.has(anchorId.value)) anchorId.value = null
  })

  const single = computed(() => (selected.value.size === 1 ? [...selected.value][0]! : null))
  const isSelected = (id: string) => selected.value.has(id)

  return { selected, single, anchor: anchorId, click, clear, isSelected }
}
