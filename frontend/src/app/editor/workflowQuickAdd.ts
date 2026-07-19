export interface WorkflowQuickAddItem {
  id: string
  kind: 'node' | 'snippet'
  title: string
  description: string
  category: string
  categoryLabel: string
  icon: string
  searchText: string
  shortcut?: string
}

export function filterWorkflowQuickAddItems(
  items: readonly WorkflowQuickAddItem[],
  query: string,
  category: string,
): WorkflowQuickAddItem[] {
  const search = query.trim().toLocaleLowerCase()
  return items
    .filter((item) =>
      search ? item.searchText.includes(search) : category === 'all' || item.category === category,
    )
    .sort((left, right) => left.title.localeCompare(right.title))
}

export function moveWorkflowQuickAddIndex(current: number, delta: number, count: number): number {
  if (!count) return 0
  return (current + delta + count) % count
}
