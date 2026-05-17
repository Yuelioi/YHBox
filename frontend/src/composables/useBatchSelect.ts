// useBatchSelect 通用列表批量选择 state。
// 用法：ContainersView / LibraryView 批量删除 UX 共用。
//   - enabled: 是否处于选择模式（true 时卡片显示 checkbox，正常点击 disable）
//   - selected: 选中 id 集合
//   - toggle(id) / clear() / selectAll(allIDs) / isSelected(id) / count
//   - enable() / disable() / toggleMode()
import { computed, ref } from 'vue'

export function useBatchSelect() {
  const enabled = ref(false)
  const selected = ref<Set<string>>(new Set())

  const count = computed(() => selected.value.size)
  const selectedIDs = computed(() => [...selected.value])

  function toggle(id: string) {
    if (selected.value.has(id)) {
      const next = new Set(selected.value)
      next.delete(id)
      selected.value = next
    } else {
      selected.value = new Set([...selected.value, id])
    }
  }

  function clear() {
    selected.value = new Set()
  }

  function selectAll(allIDs: string[]) {
    selected.value = new Set(allIDs)
  }

  function isSelected(id: string): boolean {
    return selected.value.has(id)
  }

  function enable() {
    enabled.value = true
  }
  function disable() {
    enabled.value = false
    clear()
  }
  function toggleMode() {
    if (enabled.value) disable()
    else enable()
  }

  return {
    enabled, selected, selectedIDs, count,
    toggle, clear, selectAll, isSelected,
    enable, disable, toggleMode,
  }
}
