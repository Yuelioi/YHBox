// IconBadge 纯映射: size→框/图标尺寸; color→图标色 (字面表)。
export type BadgeSize = 'sm' | 'md' | 'lg'
export type BadgeColor = 'default' | 'primary' | 'violet' | 'amber' | 'sky'

const BOX: Record<BadgeSize, string> = {
  sm: 'size-7 rounded-md',
  md: 'size-10 rounded-lg',
  lg: 'size-14 rounded-xl',
}
const ICON: Record<BadgeSize, string> = {
  sm: 'size-3.5',
  md: 'size-5',
  lg: 'size-7',
}
const COLOR: Record<BadgeColor, string> = {
  default: 'text-muted',
  primary: 'text-primary',
  violet: 'text-violet-400',
  amber: 'text-warning',
  sky: 'text-sky-400',
}

export function badgeBoxClass(size: BadgeSize): string {
  return BOX[size]
}
export function badgeIconSize(size: BadgeSize): string {
  return ICON[size]
}
export function badgeIconColor(color: BadgeColor): string {
  return COLOR[color]
}
