# Container List Dual View Design

## Goal

Improve the local container listing page so it works as a real management surface when the user has many containers.

The current page keeps large screens at three cards per row and only supports text search plus tag chips. The new design adds a shared toolbar for search, tag filtering, sorting, sort direction, view mode, pagination, and page size. It also adds a dense list view while keeping the existing card view as the default.

## Scope

In scope:

- Update `frontend/src/components/tasks/ContainersTab.vue`.
- Keep `frontend/src/views/ContainersView.vue` tab structure unchanged.
- Add card/list view switching for local containers.
- Make card layout responsive with more columns on large screens.
- Add sorting by name, created date, updated date, and node count.
- Add ascending/descending sort direction.
- Add pagination and page-size selection.
- Extend Chinese and English i18n strings under `containers`.
- Reuse the existing frontend `Container` fields: `name`, `description`, `tags`, `createdAt`, `updatedAt`, `hotkey`, and `graph.nodes`.

Out of scope:

- Backend pagination or query APIs.
- Online containers.
- Persisting filters across app restarts.
- Editing container metadata from the list page.
- A full table component abstraction shared with other pages.

## UX Design

The page keeps the existing top-level local/online tabs. Inside the local tab, controls are organized as a compact management toolbar:

- Search input: matches container name, description, tags, and category-like tag text.
- Sort select: name, created date, updated date, node count.
- Sort direction icon button: ascending or descending.
- View toggle: card or list.
- Batch select button and create button remain visible.

Tag chips stay below the toolbar because they are fast, low-friction filters. Selected tags continue to use AND semantics: a container must include every selected tag.

The default view is card view. It keeps existing run, stop, edit, delete, hotkey, status, and node-count affordances. The grid changes from fixed breakpoints to responsive tracks:

`grid-template-columns: repeat(auto-fill, minmax(260px, 1fr))`

This lets wide windows naturally show more than three cards while keeping card width readable.

List view is denser and optimized for scanning dates. Each row shows:

- Selection checkbox when batch mode is enabled.
- Name and optional description.
- Status.
- Node count.
- Created date.
- Updated date.
- Hotkey when present.
- Run/stop, edit, delete actions.

The list should be compact, but not a spreadsheet. It should use stable column widths and truncate long names/descriptions.

Pagination sits at the bottom of the local tab:

- Result count text: showing current range and total.
- `UPagination`.
- Page size select.

Initial page size: `24`, with options `12`, `24`, `48`, `96`. Search, tag filters, sort field, sort direction, view mode, or page size changes reset to page 1.

## State Model

All state can stay local to `ContainersTab.vue`:

- `search: Ref<string>`
- `selectedTags: Ref<string[]>`
- `sortKey: Ref<'name' | 'createdAt' | 'updatedAt' | 'nodes'>`
- `sortDesc: Ref<boolean>`
- `viewMode: Ref<'cards' | 'list'>`
- `page: Ref<number>`
- `pageSize: Ref<number>`

Derived data should be computed in this order:

1. `matched`: text search + selected tags.
2. `sorted`: stable copy sorted by `sortKey` and `sortDesc`.
3. `pageResult`: paginated `sorted`.
4. `visibleContainers`: `pageResult.pageItems`.

The implementation can reuse `paginate` from `frontend/src/lib/libraryFilter.ts`. If a name/date is missing, sort using an empty string. Node count is `c.graph?.nodes?.length ?? 0`.

Batch selection should operate on container IDs, as it does today. The delete count remains global to the current selection, not just the current page.

## Error And Empty States

If the store has no containers, keep the current empty state and create CTA.

If filtering produces no matches, show an empty state that makes it clear the list is filtered. It should offer a reset action that clears search and selected tags.

Deleting, running, stopping, and recording-lock behavior stay unchanged. Existing confirmation and toast behavior should be reused.

If `createdAt` or `updatedAt` is absent in older data, show a muted dash in the list view and keep sorting deterministic.

## Accessibility

Controls must use labels or `aria-label` where icon-only:

- Sort direction button.
- View mode toggle if icon-only.
- Delete buttons in cards and rows.

Interactive row actions must stop propagation so batch row/card selection does not trigger accidentally.

Both views must work with keyboard focus order: toolbar, tags, items, pagination.

## Testing

Add focused tests if there is an existing component-test pattern for `ContainersTab.vue`; otherwise verify through typecheck/build.

Minimum verification:

- `pnpm --dir frontend typecheck`
- `pnpm --dir frontend test --runInBand` is not available in this project, so use targeted Vitest only if tests are added.
- `pnpm --dir frontend build:dev`
- Manual browser verification at desktop width:
  - Card view shows more than three columns on wide screens.
  - List view displays name, node count, created date, updated date, and actions.
  - Search and tag filters affect both views.
  - Sorting by name, created date, updated date, and node count works in both directions.
  - Pagination clamps when filters reduce total pages.
  - Batch delete selection still works across view mode changes.

## Implementation Notes

Prefer small helper functions inside `ContainersTab.vue` unless the filtering/sorting logic becomes large enough to justify extraction. The current scope does not need a new global composable.

Use Nuxt UI components already present in the project: `UInput`, `USelect`, `UButton`, `UButtonGroup` or segmented buttons if available, `UPagination`, `UCheckbox`, and `StatusPill`.

Keep existing card visual language and avoid nested cards. The list view should use rows on an unframed surface, not card-per-row styling.
