export function selectLabelWidth(items: readonly unknown[], labelKey = 'label'): number {
  return flattenSelectLabels(items, labelKey).reduce(
    (longest, label) => Math.max(longest, displayWidth(label)),
    0,
  )
}

export function adaptiveSelectWidth(
  items: readonly unknown[],
  labelKey = 'label',
  minimum = 12,
  maximum = 40,
): number {
  const lower = Math.max(1, Math.min(minimum, maximum))
  const upper = Math.max(lower, maximum)
  // Nuxt UI's trigger also contains leading content, gaps, padding, and a trailing chevron.
  // Reserve enough space for that fixed chrome after measuring the longest option label.
  return Math.min(upper, Math.max(lower, selectLabelWidth(items, labelKey) + 9))
}

export function shouldUseSearchableSelect(
  items: readonly unknown[],
  searchable: boolean | 'auto' = 'auto',
): boolean {
  if (searchable !== 'auto') return searchable
  return selectItemCount(items) > 10
}

export function shouldVirtualizeSelect(items: readonly unknown[]): boolean {
  return selectItemCount(items) > 40
}

function selectItemCount(items: readonly unknown[]): number {
  let count = 0
  for (const item of items) {
    if (Array.isArray(item)) count += selectItemCount(item)
    else if (!isRecord(item) || item.type !== 'separator') count += 1
  }
  return count
}

function flattenSelectLabels(items: readonly unknown[], labelKey: string): string[] {
  const labels: string[] = []
  for (const item of items) {
    if (Array.isArray(item)) {
      labels.push(...flattenSelectLabels(item, labelKey))
      continue
    }
    if (typeof item === 'string' || typeof item === 'number' || typeof item === 'boolean') {
      labels.push(String(item))
      continue
    }
    if (!isRecord(item) || item.type === 'separator') continue
    const label = item[labelKey] ?? item.label ?? item.value
    if (typeof label === 'string' || typeof label === 'number' || typeof label === 'boolean') {
      labels.push(String(label))
    }
  }
  return labels
}

function displayWidth(value: string): number {
  let width = 0
  for (const character of value) {
    const point = character.codePointAt(0) ?? 0
    width += isWideCodePoint(point) ? 2 : 1
  }
  return width
}

function isWideCodePoint(point: number): boolean {
  return (
    point >= 0x1100 &&
    (point <= 0x115f ||
      point === 0x2329 ||
      point === 0x232a ||
      (point >= 0x2e80 && point <= 0xa4cf && point !== 0x303f) ||
      (point >= 0xac00 && point <= 0xd7a3) ||
      (point >= 0xf900 && point <= 0xfaff) ||
      (point >= 0xfe10 && point <= 0xfe19) ||
      (point >= 0xfe30 && point <= 0xfe6f) ||
      (point >= 0xff00 && point <= 0xff60) ||
      (point >= 0xffe0 && point <= 0xffe6) ||
      (point >= 0x1f300 && point <= 0x1faff) ||
      (point >= 0x20000 && point <= 0x3fffd))
  )
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
