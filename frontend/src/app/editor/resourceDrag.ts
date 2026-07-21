import type { AssetPickerSelection } from '@/stores/assets'

export const RESOURCE_DRAG_FORMAT = 'application/x-yotta-workflow-resource'

export function serializeWorkspaceResource(selection: AssetPickerSelection): string {
  return JSON.stringify(selection)
}

export function parseWorkspaceResource(raw: string): AssetPickerSelection | null {
  try {
    const value = JSON.parse(raw) as Partial<AssetPickerSelection>
    if (
      typeof value.guid !== 'string' ||
      typeof value.name !== 'string' ||
      (value.kind !== 'macro' && value.kind !== 'clip' && value.kind !== 'template') ||
      !value.blob ||
      typeof value.blob !== 'object'
    )
      return null
    return value as AssetPickerSelection
  } catch {
    return null
  }
}
