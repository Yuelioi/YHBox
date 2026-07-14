import { useLocalStorage } from '@vueuse/core'

type AssetViewMode = 'grid' | 'list'

export function useAssetBrowserPreferences<SortKey extends string>(
  kind: 'templates' | 'blueprints' | 'clips',
  defaultSort: SortKey,
) {
  const prefix = `asset.${kind}`
  return {
    query: useLocalStorage(`${prefix}.query`, ''),
    categoryFilter: useLocalStorage(`${prefix}.category`, 'all'),
    tagFilter: useLocalStorage<string[]>(`${prefix}.tags`, []),
    sortKey: useLocalStorage<SortKey>(`${prefix}.sort`, defaultSort),
    sortDesc: useLocalStorage(`${prefix}.sortDesc`, false),
    viewMode: useLocalStorage<AssetViewMode>(`${prefix}.view`, 'grid'),
  }
}
