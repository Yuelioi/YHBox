import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  add: vi.fn(),
  push: vi.fn(),
}))

vi.mock('@nuxt/ui/composables', () => ({ useToast: () => ({ add: mocks.add }) }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: mocks.push }) }))
vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

import { useRecordingStartFeedback } from './useRecordingStartFeedback'

describe('useRecordingStartFeedback', () => {
  beforeEach(() => vi.clearAllMocks())

  it('offers a direct calibration recovery action for precise recording validation', async () => {
    const { show } = useRecordingStartFeedback()

    show('workflow.recording.start_failed', {
      id: 'RECORDING_CALIBRATION_REQUIRED',
    })

    const toast = mocks.add.mock.calls[0]?.[0]
    expect(toast.description).toEqual(expect.any(String))
    expect(toast.actions).toHaveLength(1)
    expect(toast.actions[0].label).toBe('workflow.recording.open_calibration')

    await toast.actions[0].onClick()
    expect(mocks.push).toHaveBeenCalledWith({ path: '/settings', query: { section: 'input' } })
  })

  it('does not infer calibration recovery from a raw transport message', () => {
    const { show } = useRecordingStartFeedback()

    show('workflow.recording.start_failed', new Error('RECORDING_CALIBRATION_REQUIRED'))

    expect(mocks.add).toHaveBeenCalledWith(
      expect.objectContaining({
        actions: undefined,
      }),
    )
  })

  it('keeps unrelated recording failures as plain errors', () => {
    const { show } = useRecordingStartFeedback()

    show('workflow.recording.start_failed', { id: 'RECORDING_TARGET_UNAVAILABLE' })

    expect(mocks.add).toHaveBeenCalledWith(
      expect.objectContaining({ actions: undefined, color: 'error' }),
    )
  })
})
