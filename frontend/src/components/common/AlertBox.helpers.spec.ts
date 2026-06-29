import { describe, it, expect } from 'vitest'
import { alertStyle } from './AlertBox.helpers'

describe('alertStyle', () => {
  it('warning → amber box + 三角图标', () => {
    const s = alertStyle('warning')
    expect(s.box).toContain('bg-warning/10')
    expect(s.icon).toBe('text-warning')
    expect(s.defaultIcon).toBe('i-tabler-alert-triangle')
  })
  it('error → red', () => expect(alertStyle('error').icon).toBe('text-error'))
  it('info → info', () => expect(alertStyle('info').icon).toBe('text-info'))
  it('success → success', () => expect(alertStyle('success').icon).toBe('text-success'))
})
