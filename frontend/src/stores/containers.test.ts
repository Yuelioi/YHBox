import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const listMock = vi.fn(async () => [])
const deleteManyMock = vi.fn(async (_ids: string[]) => undefined)

vi.mock('@/lib/backend', () => ({
  backend: {
    containers: {
      list: () => listMock(),
      create: vi.fn(),
      update: vi.fn(),
      delete_: vi.fn(),
      deleteMany: (ids: string[]) => deleteManyMock(ids),
      exportPackage: vi.fn(),
      run: vi.fn(),
      stopAll: vi.fn(),
    },
  },
}))

vi.mock('@wailsio/runtime', () => ({
  Events: { On: vi.fn() },
}))

vi.mock('@/stores/recording', () => ({
  useRecordingStore: () => ({
    isRecording: false,
    isPaused: false,
    activeTargetContainerID: '',
  }),
}))

import { useContainersStore } from './containers'

describe('containers store batch delete', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    listMock.mockClear()
    deleteManyMock.mockClear()
  })

  it('treats successful void backend deleteMany as success', async () => {
    deleteManyMock.mockResolvedValueOnce(undefined)
    const store = useContainersStore()

    await expect(store.deleteMany(['c1'])).resolves.toBe(true)
    expect(deleteManyMock).toHaveBeenCalledWith(['c1'])
    expect(listMock).toHaveBeenCalled()
  })

  it('returns false when backend deleteMany rejects', async () => {
    deleteManyMock.mockRejectedValueOnce(new Error('failed'))
    const store = useContainersStore()

    await expect(store.deleteMany(['c1'])).resolves.toBe(false)
    expect(listMock).toHaveBeenCalled()
  })
})
