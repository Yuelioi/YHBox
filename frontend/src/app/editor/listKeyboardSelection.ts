export const NUMBERED_SELECTION_LIMIT = 9

export function moveListSelection(current: number, delta: number, count: number): number {
  if (!count) return 0
  return (current + delta + count) % count
}

export function numberedSelectionIndex(
  event: Pick<KeyboardEvent, 'key' | 'altKey' | 'ctrlKey' | 'metaKey' | 'shiftKey' | 'isComposing'>,
  count: number,
): number | undefined {
  if (
    event.isComposing ||
    event.altKey ||
    event.ctrlKey ||
    event.metaKey ||
    event.shiftKey ||
    !/^[1-9]$/.test(event.key)
  )
    return undefined
  const index = Number(event.key) - 1
  return index < count && index < NUMBERED_SELECTION_LIMIT ? index : undefined
}
