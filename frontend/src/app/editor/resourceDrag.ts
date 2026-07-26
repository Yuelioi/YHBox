export const RESOURCE_DRAG_FORMAT = 'application/x-yotta-workflow-resource'

export function serializeWorkspaceResource(guid: string): string {
  return JSON.stringify({ guid })
}

export function parseWorkspaceResource(raw: string): string | null {
  if (raw.length > 512) return null
  try {
    const value = JSON.parse(raw) as { guid?: unknown }
    if (
      typeof value.guid !== 'string' ||
      value.guid.length === 0 ||
      value.guid.length > 256 ||
      !/^[A-Za-z0-9._-]+$/.test(value.guid)
    )
      return null
    return value.guid
  } catch {
    return null
  }
}
