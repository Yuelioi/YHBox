import { ref, watch } from 'vue'

const STORAGE_KEY = 'yotta.sidebar.collapsed'

// Module-singleton state — 同步在所有 component 之间。
function readInitial(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY) === '1'
  } catch {
    return false
  }
}

const collapsed = ref<boolean>(readInitial())

watch(collapsed, (v) => {
  try {
    localStorage.setItem(STORAGE_KEY, v ? '1' : '0')
  } catch {
    /* localStorage 不可用就静默 */
  }
})

export function useSidebarCollapsed() {
  function toggle() {
    collapsed.value = !collapsed.value
  }
  function set(v: boolean) {
    collapsed.value = v
  }
  return { collapsed, toggle, set }
}
