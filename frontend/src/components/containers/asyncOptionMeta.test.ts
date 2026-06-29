import { describe, expect, it } from 'vitest'
import { applyAsyncOptionMeta } from './asyncOptionMeta'

describe('applyAsyncOptionMeta', () => {
  it('writes selected value and mapped non-empty metadata into sibling literals', () => {
    const literal = { Serial: 'old', Width: 0, Height: 0, Name: 'old device' }
    const next = applyAsyncOptionMeta(
      literal,
      'Serial',
      {
        value: 'adb-1',
        meta: { width: 1080, height: 2400, name: 'Pixel' },
      },
      { width: 'Width', height: 'Height', name: 'Name' },
    )

    expect(next).toEqual({ Serial: 'adb-1', Width: 1080, Height: 2400, Name: 'Pixel' })
    expect(literal).toEqual({ Serial: 'old', Width: 0, Height: 0, Name: 'old device' })
  })

  it('keeps existing sibling values when metadata value is empty', () => {
    const next = applyAsyncOptionMeta(
      { WebSocketURL: 'ws://old', Name: 'old tab' },
      'TargetID',
      {
        value: 'page-1',
        meta: { ws: '', name: null },
      },
      { ws: 'WebSocketURL', name: 'Name' },
    )

    expect(next).toEqual({ TargetID: 'page-1', WebSocketURL: 'ws://old', Name: 'old tab' })
  })

  it('returns null without an applyMeta contract', () => {
    expect(applyAsyncOptionMeta({}, 'Serial', { value: 'adb-1', meta: { width: 1080 } })).toBeNull()
  })
})
