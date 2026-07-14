import { describe, expect, it } from 'vitest'
import type { LauncherBlock } from '@/stores/settings'
import { cleanupStaleLauncherBlocks, filterLauncherGroups, resolveLauncher } from './launcherModel'

const blocks: LauncherBlock[] = [
  { id: 'heading', type: 'label', label: '战斗' },
  { id: 'alpha', type: 'container', containerId: 'alpha', icon: 'i-tabler-sword' },
  { id: 'missing', type: 'container', containerId: 'missing' },
  { id: 'split', type: 'hsep' },
  { id: 'beta', type: 'container', containerId: 'beta', label: '每日钓鱼' },
]

const containers = [
  { id: 'alpha', name: '深渊战斗' },
  { id: 'beta', name: '钓鱼' },
]

describe('launcher model', () => {
  it('resolves labels, legacy separators, shortcuts and stale references', () => {
    const result = resolveLauncher(blocks, containers, [
      { key: 'container.alpha', hotkeyStr: 'Ctrl+1', status: 'active' },
    ])

    expect(result.items.map((item) => item.label)).toEqual(['深渊战斗', '每日钓鱼'])
    expect(result.items[0]?.shortcut).toBe('Ctrl+1')
    expect(result.groups.map((group) => group.label)).toEqual(['战斗', ''])
    expect(result.staleBlocks.map((block) => block.id)).toEqual(['missing'])
  })

  it('filters resolved items without changing their configured groups', () => {
    const result = resolveLauncher(blocks, containers, [])

    expect(filterLauncherGroups(result.groups, 'yu')).toEqual([])
    expect(filterLauncherGroups(result.groups, '钓鱼')[0]?.items[0]?.containerId).toBe('beta')
    expect(filterLauncherGroups(result.groups, '战斗')[0]?.items[0]?.containerId).toBe('alpha')
  })

  it('removes only stale container references and preserves authored layout blocks', () => {
    const cleaned = cleanupStaleLauncherBlocks(blocks, new Set(containers.map((item) => item.id)))

    expect(cleaned.removed.map((block) => block.id)).toEqual(['missing'])
    expect(cleaned.blocks.map((block) => block.id)).toEqual(['heading', 'alpha', 'split', 'beta'])
  })
})
