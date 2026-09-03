import { describe, expect, it } from 'vitest'
import type { LauncherBlock } from '@/stores/settings'
import {
  cleanupStaleLauncherBlocks,
  filterLauncherGroups,
  normalizeLauncherSize,
  resolveLauncher,
} from './launcherModel'

const blocks: LauncherBlock[] = [
  { id: 'heading', type: 'label', label: '战斗' },
  { id: 'alpha', type: 'workflow', workflowId: 'alpha', icon: 'i-tabler-sword' },
  { id: 'vertical', type: 'vsep' },
  { id: 'missing', type: 'workflow', workflowId: 'missing' },
  { id: 'split', type: 'hsep' },
  { id: 'beta', type: 'workflow', workflowId: 'beta', label: '每日钓鱼' },
]

const workflows = [
  { workflowId: 'alpha', name: '深渊战斗' },
  { workflowId: 'beta', name: '钓鱼' },
]

describe('launcher model', () => {
  it('resolves workflow labels, separators and stale references', () => {
    const result = resolveLauncher(blocks, workflows)

    expect(result.items.map((item) => item.label)).toEqual(['深渊战斗', '每日钓鱼'])
    expect(result.items[0]?.shortcut).toBe('')
    expect(result.groups.map((group) => group.label)).toEqual(['战斗', ''])
    expect(result.staleBlocks.map((block) => block.id)).toEqual(['missing'])
    expect(result.groups[0]?.items[1]).toMatchObject({
      id: 'missing',
      stale: true,
      separatorBefore: 'vertical',
    })
  })

  it('filters resolved items without changing their configured groups', () => {
    const result = resolveLauncher(blocks, workflows)

    expect(filterLauncherGroups(result.groups, 'yu')).toEqual([])
    expect(filterLauncherGroups(result.groups, '钓鱼')[0]?.items[0]?.workflowId).toBe('beta')
    expect(filterLauncherGroups(result.groups, '战斗')[0]?.items[0]?.workflowId).toBe('alpha')
  })

  it('removes only stale workflow references and preserves authored layout blocks', () => {
    const cleaned = cleanupStaleLauncherBlocks(
      blocks,
      new Set(workflows.map((item) => item.workflowId)),
    )

    expect(cleaned.removed.map((block) => block.id)).toEqual(['missing'])
    expect(cleaned.blocks.map((block) => block.id)).toEqual([
      'heading',
      'alpha',
      'vertical',
      'split',
      'beta',
    ])
  })

  it('preserves distinct block identities for duplicate workflow entries', () => {
    const result = resolveLauncher(
      [
        { id: 'first', type: 'workflow', workflowId: 'alpha' },
        { id: 'second', type: 'workflow', workflowId: 'alpha' },
      ],
      workflows,
    )

    expect(result.items.map((item) => item.id)).toEqual(['first', 'second'])
    expect(result.items.map((item) => item.workflowId)).toEqual(['alpha', 'alpha'])
  })

  it('assigns the configured visible-launcher chord to the first nine valid items', () => {
    const manyBlocks: LauncherBlock[] = Array.from({ length: 10 }, (_, index) => ({
      id: `item-${index}`,
      type: 'workflow',
      workflowId: 'alpha',
    }))
    const result = resolveLauncher(manyBlocks, workflows, 'Ctrl+Alt')

    expect(result.items[0]?.shortcut).toBe('Ctrl+Alt+1')
    expect(result.items[8]?.shortcut).toBe('Ctrl+Alt+9')
    expect(result.items[9]?.shortcut).toBe('')
  })

  it('normalizes launcher content size to medium', () => {
    expect(normalizeLauncherSize('xsmall')).toBe('xsmall')
    expect(normalizeLauncherSize('small')).toBe('small')
    expect(normalizeLauncherSize('large')).toBe('large')
    expect(normalizeLauncherSize('xlarge')).toBe('xlarge')
    expect(normalizeLauncherSize('unknown')).toBe('medium')
  })
})
