import { describe, it, expect } from 'vitest'
import { normalizeError, errorMessage } from './invoke'

describe('normalizeError', () => {
  it('通道A validation: cause.Errors 大写', () => {
    const e = { message: 'X map[]', cause: { Errors: [{ code: 'NO_START', params: {} }] }, kind: 'RuntimeError' }
    expect(normalizeError(e)).toEqual({ errors: [{ code: 'NO_START', params: {} }] })
  })
  it('通道A apperr: cause.code 小写', () => {
    const e = { message: 'WAILS_NOT_READY', cause: { code: 'WAILS_NOT_READY', params: { x: 1 } } }
    expect(normalizeError(e)).toEqual({ code: 'WAILS_NOT_READY', params: { x: 1 } })
  })
  it('通道B worker 信封 errors 小写', () => {
    const e = { errors: [{ code: 'NO_START' }] }
    expect(normalizeError(e)).toEqual({ errors: [{ code: 'NO_START' }] })
  })
  it('通道B worker 信封 code', () => {
    const e = { code: 'WAILS_NOT_READY', params: { x: 1 } }
    expect(normalizeError(e)).toEqual({ code: 'WAILS_NOT_READY', params: { x: 1 } })
  })
  it('裸 message 回落', () => {
    expect(normalizeError(new Error('boom'))).toEqual({ message: 'boom' })
  })
  it('空/未知 → {}', () => {
    expect(normalizeError({})).toEqual({})
  })
})

describe('errorMessage', () => {
  it('validation 首条本地化 + 还有 N 个', () => {
    const e = { cause: { Errors: [{ code: 'NO_START' }, { code: 'MISSING_WINDOW_TARGET' }] } }
    const msg = errorMessage(e)
    expect(msg).toContain('Start')
    expect(msg).toMatch(/1/)
  })
  it('缺 i18n key → 回落 code 字面', () => {
    expect(errorMessage({ cause: { code: 'FOO_BAR' } })).toBe('FOO_BAR')
  })
  it('裸 message 直接显示', () => {
    expect(errorMessage(new Error('boom'))).toBe('boom')
  })
  it('{} → UNKNOWN_ERROR 文案', () => {
    const msg = errorMessage({})
    expect(msg).not.toBe('[object Object]')
    expect(msg.length).toBeGreaterThan(0)
  })
})
