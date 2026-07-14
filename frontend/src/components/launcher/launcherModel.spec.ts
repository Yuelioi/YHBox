import { describe, expect, it } from 'vitest'
import type { LauncherBlock } from '@/stores/settings'
import {
  cleanupStaleLauncherBlocks,
  countLauncherHotkeyConflicts,
  filterLauncherGroups,
  resolveLauncher,
} from './launcherModel'

const blocks: LauncherBlock[] = [
  { id: 'heading', type: 'label', label: '战斗' },
  { id: 'alpha', type: 'container', containerId: 'alpha', icon: 'i-tabler-sword' },
  { id: 'vertical', type: 'vsep' },
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
    expect(result.groups[0]?.items[1]).toMatchObject({
      id: 'missing',
      stale: true,
      separatorBefore: 'vertical',
    })
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
    expect(cleaned.blocks.map((block) => block.id)).toEqual([
      'heading',
      'alpha',
      'vertical',
      'split',
      'beta',
    ])
  })

  it('detects launcher conflicts from persisted container bindings and the live registry', () => {
    const configuredContainers = [
      { id: 'alpha', name: 'Alpha', hotkey: 'Shift + Ctrl + 1' },
      { id: 'beta', name: 'Beta', hotkey: 'ctrl+shift+1' },
      { id: 'gamma', name: 'Gamma', hotkey: 'F4' },
    ]
    const hotkeys = [
      { key: 'container.alpha', hotkeyStr: 'Shift+Ctrl+1' },
      { key: 'system.stop', hotkeyStr: 'F4' },
    ]

    expect(
      countLauncherHotkeyConflicts(
        new Set(['alpha', 'beta', 'gamma']),
        configuredContainers,
        hotkeys,
      ),
    ).toBe(3)
  })
})
