import type { LauncherBlock } from '@/stores/settings'

export type LauncherDisplay = 'both' | 'icon' | 'text'
export type LauncherSize = 'xsmall' | 'small' | 'medium' | 'large' | 'xlarge'

export interface LauncherWorkflowSummary {
  workflowId: string
  name: string
}

export interface ResolvedLauncherItem {
  id: string
  workflowId: string
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
  workflows: LauncherWorkflowSummary[],
  slotModifiers = '',
): LauncherResolution {
  const workflowsById = new Map(workflows.map((workflow) => [workflow.workflowId, workflow]))
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

    const workflow = block.workflowId ? workflowsById.get(block.workflowId) : undefined
    if (!workflow || !block.workflowId) {
      staleBlocks.push(block)
      ensureGroup(`group-${block.id}`).items.push({
        id: block.id,
        workflowId: block.workflowId ?? '',
        label: block.label?.trim() || block.workflowId || 'Unavailable',
        icon: block.icon || 'i-tabler-unlink',
        shortcut: '',
        ordinal: 0,
        stale: true,
        separatorBefore: pendingVerticalSeparator ? 'vertical' : undefined,
      })
      pendingVerticalSeparator = false
      continue
    }

    const item: ResolvedLauncherItem = {
      id: block.id,
      workflowId: block.workflowId,
      label: block.label?.trim() || workflow.name,
      icon: block.icon || 'i-tabler-player-play',
      shortcut: slotModifiers && items.length < 9 ? `${slotModifiers}+${items.length + 1}` : '',
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

export function normalizeLauncherSize(value: unknown): LauncherSize {
  return value === 'xsmall' || value === 'small' || value === 'large' || value === 'xlarge'
    ? value
    : 'medium'
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

export function cleanupStaleLauncherBlocks(blocks: LauncherBlock[], workflowIds: Set<string>) {
  const removed = blocks.filter(
    (block) =>
      block.type === 'workflow' && (!block.workflowId || !workflowIds.has(block.workflowId)),
  )
  const removedIds = new Set(removed.map((block) => block.id))
  return {
    blocks: blocks.filter((block) => !removedIds.has(block.id)),
    removed,
  }
}
