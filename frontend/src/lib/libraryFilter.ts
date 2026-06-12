// 子图库列表过滤/分组纯函数 — 抽出便于单测, LibraryExplorerModal 消费。
export interface LibraryFilterInput {
  query: string
  // null = 全部; '' = 未分类; 其余 = 精确匹配该分类
  category: string | null
  // AND: 必须含全部所选标签
  tags: string[]
}

interface FilterableSubgraph {
  label: string
  description?: string
  tags?: string[]
  category?: string
}

export function filterSubgraphs<T extends FilterableSubgraph>(items: T[], f: LibraryFilterInput): T[] {
  const q = f.query.toLowerCase().trim()
  return items.filter((item) => {
    if (f.category !== null && (item.category ?? '') !== f.category) return false
    if (f.tags.length > 0 && !f.tags.every((tg) => (item.tags ?? []).includes(tg))) return false
    if (!q) return true
    const hay = `${item.label} ${item.description ?? ''} ${(item.tags ?? []).join(' ')} ${item.category ?? ''}`.toLowerCase()
    return hay.includes(q)
  })
}

export interface CategoryGroup<T> {
  category: string
  items: T[]
}

export function groupByCategory<T extends { category?: string }>(items: T[], uncategorized: string): CategoryGroup<T>[] {
  const map = new Map<string, T[]>()
  for (const item of items) {
    const key = item.category || uncategorized
    if (!map.has(key)) map.set(key, [])
    map.get(key)!.push(item)
  }
  return Array.from(map.entries()).map(([category, items]) => ({ category, items }))
}
