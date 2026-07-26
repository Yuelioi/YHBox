export type BatchCategoryMode = 'keep' | 'set' | 'clear'
export type BatchTagMode = 'keep' | 'add' | 'remove' | 'replace' | 'clear'

export interface BatchMetadataDraft {
  categoryMode: BatchCategoryMode
  category: string
  tagMode: BatchTagMode
  tags: string[]
}

export interface OrganizedMetadata {
  category: string
  tags: string[]
}

export function createBatchMetadataDraft(): BatchMetadataDraft {
  return { categoryMode: 'keep', category: '', tagMode: 'keep', tags: [] }
}

export function hasBatchMetadataChange(draft: BatchMetadataDraft): boolean {
  if (draft.categoryMode === 'set' && !draft.category.trim()) return false
  if (
    (draft.tagMode === 'add' || draft.tagMode === 'remove' || draft.tagMode === 'replace') &&
    uniqueMetadataValues(draft.tags).length === 0
  )
    return false
  return draft.categoryMode !== 'keep' || draft.tagMode !== 'keep'
}

export function applyBatchMetadata(
  current: OrganizedMetadata,
  draft: BatchMetadataDraft,
): OrganizedMetadata {
  const currentTags = uniqueMetadataValues(current.tags)
  const draftTags = uniqueMetadataValues(draft.tags)
  let category = current.category
  if (draft.categoryMode === 'set') category = draft.category.trim()
  if (draft.categoryMode === 'clear') category = ''

  let tags = currentTags
  if (draft.tagMode === 'add') tags = uniqueMetadataValues([...currentTags, ...draftTags])
  if (draft.tagMode === 'remove') {
    const removed = new Set(draftTags.map((tag) => tag.toLocaleLowerCase()))
    tags = currentTags.filter((tag) => !removed.has(tag.toLocaleLowerCase()))
  }
  if (draft.tagMode === 'replace') tags = draftTags
  if (draft.tagMode === 'clear') tags = []

  return { category, tags }
}

export function uniqueMetadataValues(values: string[]): string[] {
  const seen = new Set<string>()
  return values
    .map((value) => value.trim())
    .filter((value) => {
      const key = value.toLocaleLowerCase()
      if (!key || seen.has(key)) return false
      seen.add(key)
      return true
    })
}
