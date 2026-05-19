import { ref, watch } from 'vue'

export const SIDEBAR_PREFS_KEY = 'yhfish.editor.sidebar'

export interface SidebarPrefs {
  leftSidebarCollapsed: boolean
  inspectorCollapsed: boolean
  favoritesExpanded: boolean
  recentExpanded: boolean
  varsExpanded: boolean
}

const DEFAULTS: SidebarPrefs = {
  leftSidebarCollapsed: false,
  inspectorCollapsed: false,
  favoritesExpanded: true,
  recentExpanded: true,
  varsExpanded: true,
}

export function useSidebarPrefs() {
  const prefs = ref<SidebarPrefs>({ ...DEFAULTS })

  function load() {
    try {
      const raw = localStorage.getItem(SIDEBAR_PREFS_KEY)
      if (raw) Object.assign(prefs.value, JSON.parse(raw))
    } catch (_e) {
      // bad JSON in localStorage → ignore, keep defaults
    }
  }

  watch(prefs, p => {
    try {
      localStorage.setItem(SIDEBAR_PREFS_KEY, JSON.stringify(p))
    } catch (_e) {
      // localStorage full / unavailable → ignore (prefs still work in-memory)
    }
  }, { deep: true })

  return { prefs, load }
}
