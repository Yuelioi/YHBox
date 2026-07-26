import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const mocks = vi.hoisted(() => ({
  revision: 2,
  changed: undefined as ((event: unknown) => void) | undefined,
  query: vi.fn(async (request: { page: number }) => ({
    items: [],
    total: 0,
    page: request.page,
    pageSize: 20,
    revision: mocks.revision,
  })),
  resolve: vi.fn(async (blob: { mediaType: string; digest: string; size: number }) => ({
    found: true,
    guid: 'asset-1',
    kind: 'template',
    name: 'Template',
    resolution: [1280, 720],
    blob,
    matchCount: 1,
  })),
}))

vi.mock('@wailsio/runtime', () => ({
  Events: {
    On: vi.fn((name: string, handler: (event: unknown) => void) => {
      if (name === 'asset:changed') mocks.changed = handler
      return () => {}
    }),
  },
}))
vi.mock('@/lib/backend', () => ({
  backend: { assets: { query: mocks.query, resolveBinding: mocks.resolve } },
}))

import { useAssetsStore } from './assets'

const query = {
  search: '',
  kind: 'template',
  category: '',
  tags: [],
  sort: 'recent_desc',
  page: 1,
  pageSize: 20,
  thumbnailBudget: 12,
  recentGUIDs: [],
}
const blob = {
  mediaType: 'image/png',
  digest: `sha256:${'a'.repeat(64)}`,
  size: 10,
}

describe('assets query store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    mocks.revision = 2
    mocks.changed = undefined
    vi.clearAllMocks()
  })

  it('shares cached pages until a revisioned asset event invalidates them', async () => {
    const store = useAssetsStore()
    await store.query(query)
    await store.query(query)
    expect(mocks.query).toHaveBeenCalledTimes(1)

    mocks.revision = 3
    mocks.changed?.({ data: [{ revision: 3 }] })
    await store.query(query)
    expect(mocks.query).toHaveBeenCalledTimes(2)
    expect(store.observedRevision).toBe(3)
  })

  it('caches exact BlobRef presentation separately from durable identity', async () => {
    const store = useAssetsStore()
    await store.resolveBinding(blob)
    await store.resolveBinding(blob)
    expect(mocks.resolve).toHaveBeenCalledTimes(1)

    mocks.changed?.({ data: [{ revision: 3 }] })
    await store.resolveBinding(blob)
    expect(mocks.resolve).toHaveBeenCalledTimes(2)
  })

  it('keeps a bounded recent-asset order for picker queries', () => {
    const store = useAssetsStore()
    for (let index = 0; index < 40; index++) store.markUsed(`asset-${index}`)
    store.markUsed('asset-10')

    expect(store.recentGUIDs).toHaveLength(32)
    expect(store.recentGUIDs[0]).toBe('asset-10')
    expect(JSON.parse(localStorage.getItem('yotta.asset-picker.recent.v1') ?? '[]')).toEqual(
      store.recentGUIDs,
    )
  })
})
