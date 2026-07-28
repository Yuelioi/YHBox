import { describe, expect, it } from 'vitest'
import { formatTimelineClock, formatTimelineDateTime, formatTimelineOffset } from './timelineTime'

describe('timeline time formatting', () => {
  it('shows millisecond precision for event clock time', () => {
    const value = '2026-07-29T01:02:03.456+08:00'
    expect(formatTimelineClock(value)).toMatch(/^\d{2}:02:03\.456$/)
    expect(formatTimelineDateTime(value)).toMatch(/^\d{4}-\d{2}-\d{2} \d{2}:02:03\.456$/)
  })

  it('shows precise offsets from Run start', () => {
    const origin = '2026-07-29T01:02:03.000Z'
    expect(formatTimelineOffset('2026-07-29T01:02:03.184Z', origin)).toBe('+184ms')
    expect(formatTimelineOffset('2026-07-29T01:02:15.406Z', origin)).toBe('+12.406s')
    expect(formatTimelineOffset('2026-07-29T01:04:06.451Z', origin)).toBe('+2m 3.451s')
  })

  it('does not invent timing when a timestamp is missing', () => {
    expect(formatTimelineClock('')).toBe('—')
    expect(formatTimelineOffset('', '')).toBe('—')
  })
})
