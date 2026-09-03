import { describe, expect, it } from 'vitest'
import { normalizeAppToast } from './useAppToast'

describe('application toast policy', () => {
  it('keeps errors visible with a close button and no countdown', () => {
    expect(normalizeAppToast({ title: 'Failed', color: 'error', duration: 1000 })).toMatchObject({
      close: true,
      duration: 0,
      progress: false,
      icon: 'i-tabler-alert-circle',
    })
  })

  it('keeps ordinary feedback transient without a progress countdown', () => {
    expect(normalizeAppToast({ title: 'Saved', color: 'success' })).toMatchObject({
      close: false,
      progress: false,
      icon: 'i-tabler-circle-check',
    })
  })
})
