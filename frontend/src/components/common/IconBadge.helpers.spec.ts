import { describe, it, expect } from 'vitest'
import { badgeBoxClass, badgeIconSize, badgeIconColor } from './IconBadge.helpers'

describe('IconBadge helpers', () => {
  it('md 框 size-10', () => expect(badgeBoxClass('md')).toContain('size-10'))
  it('lg 图标 size-7', () => expect(badgeIconSize('lg')).toBe('size-7'))
  it('primary 图标 text-primary', () => expect(badgeIconColor('primary')).toBe('text-primary'))
  it('default 图标 text-muted', () => expect(badgeIconColor('default')).toBe('text-muted'))
})
