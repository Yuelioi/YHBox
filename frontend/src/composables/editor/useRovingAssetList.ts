import { ref, watch, type Ref } from 'vue'

export function useRovingAssetList(visibleIds: Ref<string[]>) {
  const activeId = ref<string | null>(null)

  watch(
    visibleIds,
    (ids) => {
      if (!activeId.value || !ids.includes(activeId.value)) activeId.value = ids[0] ?? null
    },
    { immediate: true },
  )

  function isTabStop(id: string) {
    return activeId.value === id
  }

  function setActive(id: string) {
    activeId.value = id
  }

  function move(id: string, event: KeyboardEvent): boolean {
    const ids = visibleIds.value
    const current = Math.max(0, ids.indexOf(id))
    let target = current
    switch (event.key) {
      case 'ArrowRight':
      case 'ArrowDown':
        target = Math.min(ids.length - 1, current + 1)
        break
      case 'ArrowLeft':
      case 'ArrowUp':
        target = Math.max(0, current - 1)
        break
      case 'Home':
        target = 0
        break
      case 'End':
        target = Math.max(0, ids.length - 1)
        break
      default:
        return false
    }
    event.preventDefault()
    activeId.value = ids[target] ?? null
    const root = (event.currentTarget as HTMLElement | null)?.closest('[data-asset-browser-list]')
    const option = [...(root?.querySelectorAll<HTMLElement>('[data-asset-option]') ?? [])].find(
      (node) => node.dataset.assetId === activeId.value,
    )
    option?.focus()
    return true
  }

  return { activeId, isTabStop, setActive, move }
}
