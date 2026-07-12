// AlertBox 纯映射: type → 盒/图标 class + 默认图标 (字面表)。
export type AlertType = 'info' | 'success' | 'warning' | 'error'

interface AlertStyle {
  box: string
  icon: string
  defaultIcon: string
}

const MAP: Record<AlertType, AlertStyle> = {
  info: { box: 'bg-info/10 ring-info/20', icon: 'text-info', defaultIcon: 'i-tabler-info-circle' },
  success: {
    box: 'bg-success/10 ring-success/20',
    icon: 'text-success',
    defaultIcon: 'i-tabler-circle-check',
  },
  warning: {
    box: 'bg-warning/10 ring-warning/20',
    icon: 'text-warning',
    defaultIcon: 'i-tabler-alert-triangle',
  },
  error: {
    box: 'bg-error/10 ring-error/20',
    icon: 'text-error',
    defaultIcon: 'i-tabler-alert-circle',
  },
}

export function alertStyle(type: AlertType): AlertStyle {
  return MAP[type]
}
