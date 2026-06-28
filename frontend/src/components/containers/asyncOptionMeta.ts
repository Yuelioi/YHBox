export type AsyncOptionPayload = {
  value: unknown
  meta: Record<string, unknown>
}

export function applyAsyncOptionMeta(
  literal: Record<string, unknown>,
  pin: string,
  payload: AsyncOptionPayload,
  applyMeta?: Record<string, string>,
): Record<string, unknown> | null {
  if (!applyMeta) return null

  const next = { ...literal, [pin]: payload.value }
  for (const [metaKey, targetPin] of Object.entries(applyMeta)) {
    if (!(metaKey in payload.meta)) continue
    const value = payload.meta[metaKey]
    if (value === undefined || value === null || value === '') continue
    next[targetPin] = value
  }
  return next
}
