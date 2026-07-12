import type { Container } from '@/lib/backend'
import { paginate, type PageResult } from '@/lib/libraryFilter'

export type ContainerSortKey = 'name' | 'createdAt' | 'updatedAt' | 'nodes'

export interface ContainerFilterOptions {
  query: string
  category: string | null
  tags: string[]
}

export interface ContainerPageOptions extends ContainerFilterOptions {
  sortKey: ContainerSortKey
  sortDesc: boolean
  page: number
  pageSize: number
}

export interface ContainerPageResult extends PageResult<Container> {
  start: number
  end: number
}

export interface ContainerTagCount {
  tag: string
  count: number
}

export function containerNodeCount(c: Container): number {
  return c.graph?.nodes?.length ?? 0
}

export function containerTagsByCount(items: Container[]): ContainerTagCount[] {
  const counts: Record<string, number> = {}
  for (const c of items) {
    for (const tag of c.tags ?? []) {
      const key = tag.trim()
      if (!key) continue
      counts[key] = (counts[key] ?? 0) + 1
    }
  }
  return Object.entries(counts)
    .map(([tag, count]) => ({ tag, count }))
    .sort((a, b) => b.count - a.count || a.tag.localeCompare(b.tag))
}

export function containerCategories(items: Container[]): string[] {
  const set = new Set<string>()
  for (const c of items) {
    const key = (c.category ?? '').trim()
    if (key) set.add(key)
  }
  return [...set].sort((a, b) => a.localeCompare(b))
}

export function filterContainers(items: Container[], options: ContainerFilterOptions): Container[] {
  const q = options.query.trim().toLowerCase()
  return items.filter((c) => {
    const tags = c.tags ?? []
    if (options.category !== null && (c.category ?? '') !== options.category) return false
    if (options.tags.length > 0 && !options.tags.every((tag) => tags.includes(tag))) return false
    if (!q) return true
    const author = typeof c.author === 'string' ? c.author : (c.author?.name ?? '')
    const hay = [
      c.name ?? '',
      c.description ?? '',
      c.category ?? '',
      c.version ?? '',
      author,
      ...(c.keywords ?? []),
      ...tags,
    ]
      .join(' ')
      .toLowerCase()
    return hay.includes(q)
  })
}

export function sortContainers(
  items: Container[],
  sortKey: ContainerSortKey,
  sortDesc: boolean,
): Container[] {
  return items
    .map((item, index) => ({ item, index }))
    .sort((a, b) => {
      let cmp = 0
      switch (sortKey) {
        case 'name':
          cmp = (a.item.name ?? '').localeCompare(b.item.name ?? '')
          break
        case 'createdAt':
          cmp = (a.item.createdAt ?? '').localeCompare(b.item.createdAt ?? '')
          break
        case 'updatedAt':
          cmp = (a.item.updatedAt ?? '').localeCompare(b.item.updatedAt ?? '')
          break
        case 'nodes':
          cmp = containerNodeCount(a.item) - containerNodeCount(b.item)
          break
      }
      if (cmp === 0) return a.index - b.index
      return sortDesc ? -cmp : cmp
    })
    .map(({ item }) => item)
}

export function buildContainerPage(
  items: Container[],
  options: ContainerPageOptions,
): ContainerPageResult {
  const filtered = filterContainers(items, options)
  const sorted = sortContainers(filtered, options.sortKey, options.sortDesc)
  const result = paginate(sorted, options.page, options.pageSize)
  const start = result.total === 0 ? 0 : (result.page - 1) * options.pageSize + 1
  const end = result.total === 0 ? 0 : start + result.pageItems.length - 1
  return { ...result, start, end }
}

export function formatContainerDate(value?: string): string {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value.slice(0, 10) || '-'
  const yyyy = d.getFullYear()
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const min = String(d.getMinutes()).padStart(2, '0')
  return `${yyyy}-${mm}-${dd} ${hh}:${min}`
}
