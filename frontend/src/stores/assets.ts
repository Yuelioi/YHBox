import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { Events } from '@wailsio/runtime'
import {
  backend,
  type AssetBinding,
  type AssetPage,
  type AssetQuery,
  type BlobRef,
} from '@/lib/backend'

const RECENT_STORAGE_KEY = 'yotta.asset-picker.recent.v1'
const MAX_RECENT_ASSETS = 32
const MAX_CACHED_PAGES = 32

export interface AssetPickerSelection {
  guid: string
  kind: 'template' | 'macro' | 'clip'
  name: string
  resolution?: [number, number]
  blob: BlobRef
}

export const useAssetsStore = defineStore('assets-query', () => {
  const epoch = ref(0)
  const observedRevision = ref(0)
  const recent = ref(readRecentGUIDs())
  const pageCache = new Map<string, AssetPage>()
  const bindingCache = new Map<string, AssetBinding>()
  const inFlight = new Map<string, Promise<AssetPage>>()

  Events.On('asset:changed', (event: { data?: unknown }) => {
    const payload = Array.isArray(event.data) ? event.data[0] : event.data
    if (typeof payload === 'object' && payload !== null) {
      const revision = (payload as Record<string, unknown>).revision
      if (typeof revision === 'number') {
        observedRevision.value = Math.max(observedRevision.value, revision)
      }
    }
    invalidate()
  })

  const recentGUIDs = computed(() => [...recent.value])

  async function query(input: AssetQuery, options: { force?: boolean } = {}): Promise<AssetPage> {
    const request = normalizeQuery(input)
    const key = queryKey(request)
    if (!options.force) {
      const cached = pageCache.get(key)
      if (cached && cached.revision >= observedRevision.value) return cached
      const pending = inFlight.get(key)
      if (pending) return pending
    }
    const pending = fetchPage(request, key, 0)
    inFlight.set(key, pending)
    try {
      return await pending
    } finally {
      if (inFlight.get(key) === pending) inFlight.delete(key)
    }
  }

  async function fetchPage(request: AssetQuery, key: string, attempt: number): Promise<AssetPage> {
    const startedAtEpoch = epoch.value
    const page = await backend.assets.query(request)
    if (
      attempt === 0 &&
      (startedAtEpoch !== epoch.value || page.revision < observedRevision.value)
    ) {
      return fetchPage(request, key, 1)
    }
    observedRevision.value = Math.max(observedRevision.value, page.revision)
    rememberPage(key, page)
    return page
  }

  async function resolveBinding(blob: BlobRef): Promise<AssetBinding> {
    const key = blobKey(blob)
    const cached = bindingCache.get(key)
    if (cached) return cached
    const resolved = await backend.assets.resolveBinding(blob)
    bindingCache.set(key, resolved)
    return resolved
  }

  function markUsed(guid: string): void {
    if (!guid) return
    recent.value = [guid, ...recent.value.filter((candidate) => candidate !== guid)].slice(
      0,
      MAX_RECENT_ASSETS,
    )
    try {
      localStorage.setItem(RECENT_STORAGE_KEY, JSON.stringify(recent.value))
    } catch {
      // Recent ordering is a convenience; query and binding remain functional without storage.
    }
  }

  function invalidate(): void {
    epoch.value += 1
    pageCache.clear()
    bindingCache.clear()
  }

  function rememberPage(key: string, page: AssetPage): void {
    pageCache.delete(key)
    pageCache.set(key, page)
    while (pageCache.size > MAX_CACHED_PAGES) {
      const oldest = pageCache.keys().next().value
      if (typeof oldest !== 'string') break
      pageCache.delete(oldest)
    }
  }

  return { epoch, observedRevision, recentGUIDs, query, resolveBinding, markUsed, invalidate }
})

function normalizeQuery(query: AssetQuery): AssetQuery {
  return {
    search: query.search.trim(),
    kind: query.kind,
    category: query.category.trim(),
    tags: query.tags.map((tag) => tag.trim()).filter(Boolean),
    sort: query.sort,
    page: Math.max(1, Math.trunc(query.page)),
    pageSize: Math.max(1, Math.trunc(query.pageSize)),
    thumbnailBudget: Math.max(0, Math.trunc(query.thumbnailBudget)),
    recentGUIDs: [...query.recentGUIDs],
  }
}

function queryKey(query: AssetQuery): string {
  return JSON.stringify(query)
}

function blobKey(blob: BlobRef): string {
  return `${blob.mediaType}\u0000${blob.digest}\u0000${blob.size}`
}

function readRecentGUIDs(): string[] {
  try {
    const parsed: unknown = JSON.parse(localStorage.getItem(RECENT_STORAGE_KEY) ?? '[]')
    if (!Array.isArray(parsed)) return []
    return parsed
      .filter((value): value is string => typeof value === 'string' && value.length > 0)
      .slice(0, MAX_RECENT_ASSETS)
  } catch {
    return []
  }
}
