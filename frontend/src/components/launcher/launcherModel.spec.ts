import { describe, expect, it } from 'vitest'
import type { LauncherBlock } from '@/stores/settings'
import { cleanupStaleLauncherBlocks, filterLauncherGroups, resolveLauncher } from './launcherModel'

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
})
