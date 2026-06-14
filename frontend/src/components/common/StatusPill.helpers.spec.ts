import { describe, it, expect } from 'vitest'
import { statusPillClass } from './StatusPill.helpers'

describe('statusPillClass', () => {
  it('online → primary tint', () => expect(statusPillClass('online')).toContain('text-primary'))
  it('ready → muted', () => expect(statusPillClass('ready')).toContain('text-muted'))
  it('paused → warning', () => expect(statusPillClass('paused')).toContain('text-warning'))
  it('failed → error', () => expect(statusPillClass('failed')).toContain('text-error'))
})
