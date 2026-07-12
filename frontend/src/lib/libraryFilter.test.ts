import { describe, expect, it } from 'vitest'
import { filterSubgraphs, groupByCategory, paginate } from './libraryFilter'

const items = [
  { label: '钓鱼主循环', description: '抛竿收竿', tags: ['钓鱼', '主流程'], category: '钓鱼' },
  { label: '上钩检测', tags: ['钓鱼'], category: '钓鱼' },
  { label: '通用点击', description: 'click helper', tags: ['工具'] },
  { label: '空白', tags: [] },
]

describe('filterSubgraphs', () => {
  it('无过滤条件时原样返回', () => {
    expect(filterSubgraphs(items, { query: '', category: null, tags: [] })).toHaveLength(4)
  })
  it('query 匹配 label/description/tags/category, 大小写不敏感', () => {
    expect(filterSubgraphs(items, { query: 'CLICK', category: null, tags: [] })).toHaveLength(1)
    expect(filterSubgraphs(items, { query: '钓鱼', category: null, tags: [] })).toHaveLength(2)
  })
  it('category 精确匹配; 空串 = 未分类', () => {
    expect(filterSubgraphs(items, { query: '', category: '钓鱼', tags: [] })).toHaveLength(2)
    expect(filterSubgraphs(items, { query: '', category: '', tags: [] })).toHaveLength(2)
  })
  it('tags 为 AND 语义', () => {
    expect(filterSubgraphs(items, { query: '', category: null, tags: ['钓鱼'] })).toHaveLength(2)
    expect(
      filterSubgraphs(items, { query: '', category: null, tags: ['钓鱼', '主流程'] }),
    ).toHaveLength(1)
  })
  it('三条件叠加', () => {
    expect(
      filterSubgraphs(items, { query: '上钩', category: '钓鱼', tags: ['钓鱼'] }),
    ).toHaveLength(1)
  })
})

describe('groupByCategory', () => {
  it('按 category 分组, 空归未分类, 组序 = 首现序', () => {
    const groups = groupByCategory(items, '未分类')
    expect(groups.map((g) => g.category)).toEqual(['钓鱼', '未分类'])
    expect(groups[0].items).toHaveLength(2)
    expect(groups[1].items).toHaveLength(2)
  })
})

describe('paginate', () => {
  const nums = Array.from({ length: 23 }, (_, i) => i + 1)
  it('切页 + 总数 + 总页数', () => {
    const r = paginate(nums, 1, 10)
    expect(r.pageItems).toEqual([1, 2, 3, 4, 5, 6, 7, 8, 9, 10])
    expect(r.total).toBe(23)
    expect(r.totalPages).toBe(3)
  })
  it('末页不足一页只给剩余', () => {
    expect(paginate(nums, 3, 10).pageItems).toEqual([21, 22, 23])
  })
  it('页码越界钳制 (上下界)', () => {
    expect(paginate(nums, 99, 10).page).toBe(3)
    expect(paginate(nums, 0, 10).page).toBe(1)
  })
  it('空列表 = 1 页 0 项', () => {
    const r = paginate([], 5, 20)
    expect(r.pageItems).toEqual([])
    expect(r.totalPages).toBe(1)
    expect(r.page).toBe(1)
  })
})
