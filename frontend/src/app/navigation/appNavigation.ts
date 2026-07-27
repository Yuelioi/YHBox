export interface AppNavigationItem {
  key: 'workflows' | 'assets' | 'schedules'
  to: string
  icon: string
  label: string
  active: boolean
}

export interface AppNavigationModel {
  primary: AppNavigationItem[]
  contextTitle: string
  contextIcon: string
}

type Translate = (key: string) => string

const destinations = [
  {
    key: 'workflows',
    route: 'workflows',
    to: '/workflows',
    icon: 'i-tabler-route',
    labelKey: 'sidebar.workflows',
  },
  {
    key: 'assets',
    route: 'assets',
    to: '/assets',
    icon: 'i-tabler-library',
    labelKey: 'sidebar.assets',
  },
  {
    key: 'schedules',
    route: 'schedules',
    to: '/schedules',
    icon: 'i-tabler-clock',
    labelKey: 'sidebar.schedules',
  },
] as const

const utilityContexts: Record<string, { titleKey: string; icon: string }> = {
  settings: { titleKey: 'sidebar.settings', icon: 'i-tabler-settings' },
  about: { titleKey: 'sidebar.about', icon: 'i-tabler-info-circle' },
}

export function buildAppNavigation(routeName: string, translate: Translate): AppNavigationModel {
  const activePrimary = routeName === 'workflow-edit' ? 'workflows' : routeName
  const context = utilityContexts[routeName]
  return {
    primary: destinations.map((item) => ({
      key: item.key,
      to: item.to,
      icon: item.icon,
      label: translate(item.labelKey),
      active: activePrimary === item.route,
    })),
    contextTitle: context ? translate(context.titleKey) : '',
    contextIcon: context?.icon ?? '',
  }
}
