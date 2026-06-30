import { describe, expect, it } from 'vitest'
import type { Container } from '@/lib/backend'
import { buildContainerPage, containerCategories, containerTagsByCount, filterContainers, formatContainerDate, sortContainers } from '@/lib/containerList'

function c(partial: Partial<Container>): Container {
  return {
    schemaVersion: 1,
    id: partial.id ?? 'id',
    name: partial.name ?? '',
    description: partial.description,
    tags: partial.tags,
    category: partial.category,
    keywords: partial.keywords,
    version: partial.version,
    author: partial.author,
    hotkey: partial.hotkey,
    graph: partial.graph ?? { id: 'g', schemaVersion: 1, nodes: [], edges: [] } as any,
    createdAt: partial.createdAt ?? '',
    updatedAt: partial.updatedAt ?? '',
  } as Container
}

describe('filterContainers', () => {
  it('matches name, description, and tags with AND tag filters', () => {
    const items = [
      c({ id: 'a', name: 'Fishing Daily', description: 'lake route', tags: ['daily', 'fish'] }),
      c({ id: 'b', name: 'Raid', description: 'boss', tags: ['daily', 'raid'] }),
    ]

    expect(filterContainers(items, { query: 'lake', category: null, tags: ['daily'] }).map((x) => x.id)).toEqual(['a'])
    expect(filterContainers(items, { query: 'raid', category: null, tags: ['fish'] }).map((x) => x.id)).toEqual([])
  })

  it('matches category and package-backed fields', () => {
    const items = [
      c({ id: 'a', name: 'Fishing Daily', category: 'daily', keywords: ['lake'], version: '1.2.0', author: { name: 'yl' } }),
      c({ id: 'b', name: 'Raid', category: 'combat', keywords: ['boss'], version: '0.4.0', author: { name: 'team' } }),
    ]

    expect(filterContainers(items, { query: 'yl', category: 'daily', tags: [] }).map((x) => x.id)).toEqual(['a'])
    expect(filterContainers(items, { query: 'boss', category: 'daily', tags: [] }).map((x) => x.id)).toEqual([])
  })
})

describe('containerCategories', () => {
  it('returns sorted non-empty categories', () => {
    const items = [
      c({ category: 'combat' }),
      c({ category: '' }),
      c({ category: 'daily' }),
      c({ category: 'combat' }),
    ]

    expect(containerCategories(items)).toEqual(['combat', 'daily'])
  })
})

describe('containerTagsByCount', () => {
  it('returns tags ordered by usage count then name', () => {
    const items = [
      c({ id: 'a', tags: ['daily', 'fish'] }),
      c({ id: 'b', tags: ['daily', 'raid'] }),
      c({ id: 'c', tags: ['fish'] }),
    ]

    expect(containerTagsByCount(items)).toEqual([
      { tag: 'daily', count: 2 },
      { tag: 'fish', count: 2 },
      { tag: 'raid', count: 1 },
    ])
  })
})

describe('sortContainers', () => {
  it('sorts by updated date descending and keeps stable order for ties', () => {
    const items = [
      c({ id: 'a', updatedAt: '2026-06-01T00:00:00Z' }),
      c({ id: 'b', updatedAt: '2026-06-03T00:00:00Z' }),
      c({ id: 'c', updatedAt: '2026-06-03T00:00:00Z' }),
    ]

    expect(sortContainers(items, 'updatedAt', true).map((x) => x.id)).toEqual(['b', 'c', 'a'])
  })

  it('sorts by node count', () => {
    const items = [
      c({ id: 'small', graph: { id: 'g1', schemaVersion: 1, nodes: [{}], edges: [] } as any }),
      c({ id: 'large', graph: { id: 'g2', schemaVersion: 1, nodes: [{}, {}, {}], edges: [] } as any }),
    ]

    expect(sortContainers(items, 'nodes', true).map((x) => x.id)).toEqual(['large', 'small'])
  })
})

describe('buildContainerPage', () => {
  it('returns clamped page items and visible range', () => {
    const items = Array.from({ length: 5 }, (_, i) => c({ id: String(i), name: `c${i}` }))
    const page = buildContainerPage(items, { query: '', category: null, tags: [], sortKey: 'name', sortDesc: false, page: 3, pageSize: 2 })

    expect(page.page).toBe(3)
    expect(page.pageItems.map((x) => x.id)).toEqual(['4'])
    expect(page.start).toBe(5)
    expect(page.end).toBe(5)
  })
})

describe('formatContainerDate', () => {
  it('returns a dash for empty dates and a compact value for valid dates', () => {
    expect(formatContainerDate('')).toBe('-')
    expect(formatContainerDate('2026-06-30T10:20:00Z')).toContain('2026')
  })
})
