import type {
  WorkflowResource,
  YottaWorkflowSource,
} from '../../../../contracts/workflow/current/workflow-source'

export interface WorkflowResourceQuery {
  search: string
  category: string
  allCategoriesValue: string
  tags: string[]
  sort: 'name_asc' | 'name_desc'
  page: number
  pageSize: number
}

export interface WorkflowResourcePage {
  items: WorkflowResource[]
  total: number
  categories: Array<{ value: string; count: number }>
  tags: Array<{ value: string; count: number }>
}

export function projectWorkflowResourcePage(
  resources: readonly WorkflowResource[],
  query: WorkflowResourceQuery,
): WorkflowResourcePage {
  const search = query.search.trim().toLocaleLowerCase()
  const filtered = resources.filter((resource) => {
    if (
      search &&
      ![
        resource.id,
        resource.name,
        resource.description,
        resource.category,
        ...(resource.tags ?? []),
      ]
        .filter((value): value is string => Boolean(value))
        .some((value) => value.toLocaleLowerCase().includes(search))
    )
      return false
    if (query.category !== query.allCategoriesValue && resource.category !== query.category)
      return false
    return query.tags.every((tag) => resource.tags?.includes(tag))
  })
  filtered.sort((left, right) => {
    const order = left.name.localeCompare(right.name)
    return query.sort === 'name_desc' ? -order : order
  })
  const page = Math.max(1, Math.trunc(query.page))
  const pageSize = Math.max(1, Math.trunc(query.pageSize))
  const start = (page - 1) * pageSize
  return {
    items: filtered.slice(start, start + pageSize),
    total: filtered.length,
    categories: countValues(resources.map((resource) => resource.category)),
    tags: countValues(resources.flatMap((resource) => resource.tags ?? [])),
  }
}

export function workflowResourceReferenceCount(
  source: Pick<YottaWorkflowSource, 'graphs'>,
  resourceId: string,
): number {
  let count = 0
  for (const graph of source.graphs) {
    for (const owner of [...graph.nodes, ...(graph.calls ?? [])]) {
      for (const binding of Object.values(owner.bindings)) {
        if (binding.kind === 'resource' && binding.resource?.resourceId === resourceId) count++
      }
    }
  }
  return count
}

function countValues(values: Array<string | undefined>): Array<{ value: string; count: number }> {
  const counts = new Map<string, number>()
  for (const value of values) {
    if (value) counts.set(value, (counts.get(value) ?? 0) + 1)
  }
  return [...counts]
    .map(([value, count]) => ({ value, count }))
    .sort((left, right) => left.value.localeCompare(right.value))
}
