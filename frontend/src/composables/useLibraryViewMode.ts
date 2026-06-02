// useLibraryViewMode 库视图模式（grid / list）持久化到 localStorage。
// 走 module-singleton state，LibraryView 主程序和潜在的 NodePalette mini 可以共享。
import { ref, watch } from 'vue'

const STORAGE_KEY = 'yotta.library.viewMode'

export type LibraryViewMode = 'grid' | 'list'

function readInitial(): LibraryViewMode {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    return v === 'list' ? 'list' : 'grid'
  } catch {
    return 'grid'
  }
}

const mode = ref<LibraryViewMode>(readInitial())

watch(mode, (v) => {
  try {
    localStorage.setItem(STORAGE_KEY, v)
  } catch {
    /* localStorage 不可用就静默 */
  }
})

export function useLibraryViewMode() {
  function set(v: LibraryViewMode) {
    mode.value = v
  }
  function toggle() {
    mode.value = mode.value === 'grid' ? 'list' : 'grid'
  }
  return { mode, set, toggle }
}
