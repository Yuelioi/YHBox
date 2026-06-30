# Container List Dual View Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the approved dual-view local containers page with shared search, tag filtering, sorting, pagination, and responsive cards.

**Architecture:** Extract container list filtering/sorting/pagination into `frontend/src/lib/containerList.ts` with focused Vitest coverage. Update `ContainersTab.vue` to render card and list views from the same computed page result, keeping existing run/edit/delete/batch behavior intact.

**Tech Stack:** Vue 3.5, Pinia, Nuxt UI v4 components, Vitest, existing `paginate` helper from `frontend/src/lib/libraryFilter.ts`.

---

## File Map

- Create `frontend/src/lib/containerList.ts`: pure list helpers for query matching, tag AND filtering, stable sorting, date display, and paginated ranges.
- Create `frontend/src/lib/containerList.test.ts`: Vitest coverage for helper behavior.
- Modify `frontend/src/components/tasks/ContainersTab.vue`: toolbar, dual view render branches, pagination footer, reset-filter empty state.
- Modify `frontend/src/i18n/zh.ts`: Chinese labels for sort, view mode, pagination, list headers, filtered empty state.
- Modify `frontend/src/i18n/en.ts`: English labels matching the Chinese keys.

## Task 1: Pure List Helpers

**Files:**

- Create: `frontend/src/lib/containerList.ts`
- Test: `frontend/src/lib/containerList.test.ts`

- [x] **Step 1: Write helper tests**

Create `frontend/src/lib/containerList.test.ts` with tests that cover:

```ts
import { describe, expect, it } from 'vitest'
import type { Container } from '@/lib/backend'
import { buildContainerPage, filterContainers, formatContainerDate, sortContainers } from '@/lib/containerList'

function c(partial: Partial<Container>): Container {
  return {
    schemaVersion: 1,
    id: partial.id ?? 'id',
    name: partial.name ?? '',
    description: partial.description,
    tags: partial.tags,
    hotkey: partial.hotkey,
    graph: partial.graph ?? { version: 1, nodes: [], edges: [] } as any,
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

    expect(filterContainers(items, { query: 'lake', tags: ['daily'] }).map((x) => x.id)).toEqual(['a'])
    expect(filterContainers(items, { query: 'raid', tags: ['fish'] }).map((x) => x.id)).toEqual([])
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
      c({ id: 'small', graph: { version: 1, nodes: [{}], edges: [] } as any }),
      c({ id: 'large', graph: { version: 1, nodes: [{}, {}, {}], edges: [] } as any }),
    ]

    expect(sortContainers(items, 'nodes', true).map((x) => x.id)).toEqual(['large', 'small'])
  })
})

describe('buildContainerPage', () => {
  it('returns clamped page items and visible range', () => {
    const items = Array.from({ length: 5 }, (_, i) => c({ id: String(i), name: `c${i}` }))
    const page = buildContainerPage(items, { query: '', tags: [], sortKey: 'name', sortDesc: false, page: 3, pageSize: 2 })

    expect(page.page).toBe(3)
    expect(page.pageItems.map((x) => x.id)).toEqual(['4'])
    expect(page.start).toBe(5)
    expect(page.end).toBe(5)
  })
})

describe('formatContainerDate', () => {
  it('returns a dash for empty dates and a compact value for valid dates', () => {
    expect(formatContainerDate('')).toBe('—')
    expect(formatContainerDate('2026-06-30T10:20:00Z')).toContain('2026')
  })
})
```

- [x] **Step 2: Run tests and confirm failure**

Run:

```bash
pnpm --dir frontend exec vitest run src/lib/containerList.test.ts
```

Expected: fails because `@/lib/containerList` does not exist.

- [x] **Step 3: Implement helpers**

Create `frontend/src/lib/containerList.ts` with exported types and functions used by the tests:

```ts
import type { Container } from '@/lib/backend'
import { paginate, type PageResult } from '@/lib/libraryFilter'

export type ContainerSortKey = 'name' | 'createdAt' | 'updatedAt' | 'nodes'

export interface ContainerFilterOptions {
  query: string
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

export function containerNodeCount(c: Container): number {
  return c.graph?.nodes?.length ?? 0
}

export function filterContainers(items: Container[], options: ContainerFilterOptions): Container[] {
  const q = options.query.trim().toLowerCase()
  return items.filter((c) => {
    const tags = c.tags ?? []
    if (options.tags.length > 0 && !options.tags.every((tag) => tags.includes(tag))) return false
    if (!q) return true
    const hay = `${c.name ?? ''} ${c.description ?? ''} ${tags.join(' ')}`.toLowerCase()
    return hay.includes(q)
  })
}

export function sortContainers(items: Container[], sortKey: ContainerSortKey, sortDesc: boolean): Container[] {
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
      if (cmp === 0) cmp = a.index - b.index
      return sortDesc ? -cmp : cmp
    })
    .map(({ item }) => item)
}

export function buildContainerPage(items: Container[], options: ContainerPageOptions): ContainerPageResult {
  const filtered = filterContainers(items, options)
  const sorted = sortContainers(filtered, options.sortKey, options.sortDesc)
  const result = paginate(sorted, options.page, options.pageSize)
  const start = result.total === 0 ? 0 : (result.page - 1) * options.pageSize + 1
  const end = result.total === 0 ? 0 : start + result.pageItems.length - 1
  return { ...result, start, end }
}

export function formatContainerDate(value?: string): string {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value.slice(0, 10) || '—'
  const yyyy = d.getFullYear()
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const min = String(d.getMinutes()).padStart(2, '0')
  return `${yyyy}-${mm}-${dd} ${hh}:${min}`
}
```

- [x] **Step 4: Run helper tests**

Run:

```bash
pnpm --dir frontend exec vitest run src/lib/containerList.test.ts
```

Expected: PASS.

## Task 2: ContainersTab UI

**Files:**

- Modify: `frontend/src/components/tasks/ContainersTab.vue`
- Modify: `frontend/src/i18n/zh.ts`
- Modify: `frontend/src/i18n/en.ts`

- [x] **Step 1: Update script state**

Add imports from `@/lib/containerList`, add `sortKey`, `sortDesc`, `viewMode`, `page`, `pageSize`, `pageResult`, `visibleContainers`, `resetFilters`, `formatDate`, and watchers that reset to page 1 when filters/sort/view/page size changes.

- [x] **Step 2: Update template**

Replace the fixed header with toolbar controls, keep tag chips, render the no-containers and no-match empty states separately, render cards from `visibleContainers`, add a list-view branch, and add the pagination footer.

- [x] **Step 3: Update i18n**

Add `containers.sort`, `containers.view`, `containers.pagination`, `containers.no_match_*`, and list header labels in both `zh.ts` and `en.ts`.

- [x] **Step 4: Run frontend typecheck**

Run:

```bash
pnpm --dir frontend typecheck
```

Expected: PASS.

## Task 3: Build Verification And Commit

- [x] **Step 1: Run targeted tests**

Run:

```bash
pnpm --dir frontend exec vitest run src/lib/containerList.test.ts
```

Expected: PASS.

- [x] **Step 2: Run dev build**

Run:

```bash
pnpm --dir frontend build:dev
```

Expected: PASS.

- [x] **Step 3: Commit implementation**

Run:

```bash
git add frontend/src/lib/containerList.ts frontend/src/lib/containerList.test.ts frontend/src/components/tasks/ContainersTab.vue frontend/src/i18n/zh.ts frontend/src/i18n/en.ts docs/superpowers/plans/2026-06-30-container-list-dual-view.md
git commit -m "feat(containers): add dual view list controls"
```
