import { describe, expect, it } from 'vitest'
import {
  filterWorkflowQuickAddItems,
  moveWorkflowQuickAddIndex,
  type WorkflowQuickAddItem,
} from './workflowQuickAdd'

const items: WorkflowQuickAddItem[] = [
  {
    id: 'click',
    kind: 'node',
    title: '点击模板',
    description: '视觉',
    category: 'vision',
    categoryLabel: '视觉',
    icon: 'i-tabler-photo',
    searchText: '点击模板 vision click-template',
  },
  {
    id: 'snippet',
    kind: 'snippet',
    title: '登录按钮',
    description: '复用节点',
    category: 'snippets',
    categoryLabel: 'Snippets',
    icon: 'i-tabler-bookmark',
    searchText: '登录按钮 复用节点 snippet',
  },
]

describe('workflow quick add', () => {
  it('uses categories when blank and searches across every category', () => {
    expect(filterWorkflowQuickAddItems(items, '', 'vision').map((item) => item.id)).toEqual([
      'click',
    ])
    expect(filterWorkflowQuickAddItems(items, 'snippet', 'vision').map((item) => item.id)).toEqual([
      'snippet',
    ])
  })

  it('wraps keyboard selection', () => {
    expect(moveWorkflowQuickAddIndex(0, -1, 2)).toBe(1)
    expect(moveWorkflowQuickAddIndex(1, 1, 2)).toBe(0)
  })
})
