import type { LauncherBlock } from '@/stores/settings'

export type LauncherDisplay = 'both' | 'icon' | 'text'

export interface LauncherContainerSummary {
  id: string
  name: string
  hotkey?: string
}

export interface LauncherHotkeySummary {
  key: string
  hotkeyStr: string
  status?: string
  lastError?: string
}

export interface ResolvedLauncherItem {
  id: string
  containerId: string
  label: string
  icon: string
  shortcut: string
  ordinal: number
  stale?: boolean
  separatorBefore?: 'vertical'
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
      .map((entry) => [containerIdFromHotkeyKey(entry.key), entry.hotkeyStr] as const)
      .filter(([containerId]) => !!containerId),
  )
  const groups: ResolvedLauncherGroup[] = []
  const items: ResolvedLauncherItem[] = []
  const staleBlocks: LauncherBlock[] = []
  let current: ResolvedLauncherGroup | null = null
  let pendingVerticalSeparator = false

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
      pendingVerticalSeparator = false
      continue
    }
    if (block.type === 'hsep') {
      current = null
      pendingVerticalSeparator = false
      continue
    }
    if (block.type === 'vsep') {
      pendingVerticalSeparator = true
      continue
    }

    const container = block.containerId ? containersById.get(block.containerId) : undefined
    if (!container || !block.containerId) {
      staleBlocks.push(block)
      ensureGroup(`group-${block.id}`).items.push({
        id: block.id,
        containerId: block.containerId ?? '',
        label: block.label?.trim() || block.containerId || 'Unavailable',
        icon: block.icon || 'i-tabler-unlink',
        shortcut: block.containerId ? hotkeysById.get(block.containerId) || '' : '',
        ordinal: 0,
        stale: true,
        separatorBefore: pendingVerticalSeparator ? 'vertical' : undefined,
      })
      pendingVerticalSeparator = false
      continue
    }

    const item: ResolvedLauncherItem = {
      id: block.id,
      containerId: block.containerId,
      label: block.label?.trim() || container.name,
      icon: block.icon || 'i-tabler-player-play',
      shortcut: hotkeysById.get(block.containerId) || '',
      ordinal: items.length + 1,
      separatorBefore: pendingVerticalSeparator ? 'vertical' : undefined,
    }
    ensureGroup(`group-${block.id}`).items.push(item)
    items.push(item)
    pendingVerticalSeparator = false
  }

  return {
    groups: groups.filter((group) => group.items.length > 0),
    items,
    staleBlocks,
  }
}

export function normalizeLauncherDisplay(value: unknown): LauncherDisplay {
  return value === 'icon' || value === 'text' ? value : 'both'
}

export function containerHotkeyKey(containerId: string) {
  return `${CONTAINER_HOTKEY_PREFIX}${containerId}`
}

export function containerIdFromHotkeyKey(key: string) {
  return key.startsWith(CONTAINER_HOTKEY_PREFIX) ? key.slice(CONTAINER_HOTKEY_PREFIX.length) : ''
}

const CONTAINER_HOTKEY_PREFIX = 'container.'

export function countLauncherHotkeyConflicts(
  launcherContainerIds: Set<string>,
  containers: LauncherContainerSummary[],
  hotkeys: LauncherHotkeySummary[],
) {
  const bindings = new Map(hotkeys.map((entry) => [entry.key, entry.hotkeyStr]))
  for (const container of containers) {
    if (container.hotkey) bindings.set(containerHotkeyKey(container.id), container.hotkey)
  }

  const normalizedCounts = new Map<string, number>()
  for (const hotkey of bindings.values()) {
    const normalized = normalizeHotkeyForHealth(hotkey)
    if (normalized) normalizedCounts.set(normalized, (normalizedCounts.get(normalized) ?? 0) + 1)
  }

  return containers.filter((container) => {
    if (!launcherContainerIds.has(container.id)) return false
    const configured = normalizeHotkeyForHealth(container.hotkey ?? '')
    const registryEntry = hotkeys.find((entry) => entry.key === containerHotkeyKey(container.id))
    return (
      (!!configured && (normalizedCounts.get(configured) ?? 0) > 1) ||
      registryEntry?.lastError?.startsWith('[conflict]')
    )
  }).length
}

function normalizeHotkeyForHealth(value: string) {
  const modifierOrder = new Map([
    ['ctrl', 0],
    ['control', 0],
    ['alt', 1],
    ['shift', 2],
    ['win', 3],
    ['meta', 3],
  ])
  const tokens = value
    .split('+')
    .map((token) => token.trim().toLocaleLowerCase())
    .filter(Boolean)
  if (!tokens.length) return ''
  return tokens
    .sort((left, right) => {
      const leftOrder = modifierOrder.get(left) ?? 10
      const rightOrder = modifierOrder.get(right) ?? 10
      return leftOrder - rightOrder || left.localeCompare(right)
    })
    .join('+')
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
