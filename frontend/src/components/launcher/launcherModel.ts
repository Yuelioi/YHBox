import type { LauncherBlock } from '@/stores/settings'

export interface LauncherContainerSummary {
  id: string
  name: string
}

export interface LauncherHotkeySummary {
  key: string
  hotkeyStr: string
  status?: string
}

export interface ResolvedLauncherItem {
  id: string
  containerId: string
  label: string
  icon: string
  shortcut: string
  ordinal: number
}

export interface ResolvedLauncherGroup {
  id: string
  label: string
  items: ResolvedLauncherItem[]
}

export interface LauncherResolution {
  groups: ResolvedLauncherGroup[]
  items: ResolvedLauncherItem[]
  staleBlocks: LauncherBlock[]
}

export function resolveLauncher(
  blocks: LauncherBlock[],
  containers: LauncherContainerSummary[],
  hotkeys: LauncherHotkeySummary[],
): LauncherResolution {
  const containersById = new Map(containers.map((container) => [container.id, container]))
  const hotkeysById = new Map(
    hotkeys
      .filter((entry) => entry.key.startsWith('container.'))
      .map((entry) => [entry.key.slice('container.'.length), entry.hotkeyStr]),
  )
  const groups: ResolvedLauncherGroup[] = []
  const items: ResolvedLauncherItem[] = []
  const staleBlocks: LauncherBlock[] = []
  let current: ResolvedLauncherGroup | null = null

  const ensureGroup = (id: string, label = '') => {
    if (!current) {
      current = { id, label, items: [] }
      groups.push(current)
    }
    return current
  }

  for (const block of blocks) {
    if (block.type === 'label') {
      current = { id: block.id, label: block.label?.trim() ?? '', items: [] }
      groups.push(current)
      continue
    }
    if (block.type === 'hsep' || block.type === 'vsep') {
      current = null
      continue
    }

    const container = block.containerId ? containersById.get(block.containerId) : undefined
    if (!container || !block.containerId) {
      staleBlocks.push(block)
      continue
    }

    const item: ResolvedLauncherItem = {
      id: block.id,
      containerId: block.containerId,
      label: block.label?.trim() || container.name,
      icon: block.icon || 'i-tabler-player-play',
      shortcut: hotkeysById.get(block.containerId) || '',
      ordinal: items.length + 1,
    }
    ensureGroup(`group-${block.id}`).items.push(item)
    items.push(item)
  }

  return {
    groups: groups.filter((group) => group.items.length > 0),
    items,
    staleBlocks,
  }
}

export function filterLauncherGroups(
  groups: ResolvedLauncherGroup[],
  rawQuery: string,
): ResolvedLauncherGroup[] {
  const query = rawQuery.trim().toLocaleLowerCase()
  if (!query) return groups

  return groups
    .map((group) => {
      const groupMatches = group.label.toLocaleLowerCase().includes(query)
      return {
        ...group,
        items: groupMatches
          ? group.items
          : group.items.filter((item) => item.label.toLocaleLowerCase().includes(query)),
      }
    })
    .filter((group) => group.items.length > 0)
}

export function cleanupStaleLauncherBlocks(blocks: LauncherBlock[], containerIds: Set<string>) {
  const removed = blocks.filter(
    (block) =>
      block.type === 'container' && (!block.containerId || !containerIds.has(block.containerId)),
  )
  const removedIds = new Set(removed.map((block) => block.id))
  return {
    blocks: blocks.filter((block) => !removedIds.has(block.id)),
    removed,
  }
}
