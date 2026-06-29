export type AsyncOption = {
  value: unknown
  label?: string
  meta?: Record<string, unknown>
}

export function normalizeAsyncDropdownValue(v: unknown): unknown {
  if (v && typeof v === 'object' && 'value' in v) {
    return (v as { value: unknown }).value
  }
  return v
}

export function asyncOptionPayloadForValue(
  options: AsyncOption[],
  value: unknown,
): { value: unknown; meta: Record<string, unknown> } | null {
  const selected = options.find((o) => String(o.value ?? '') === String(value ?? ''))
  if (!selected?.meta || Object.keys(selected.meta).length === 0) return null
  return { value, meta: selected.meta }
}
