// frontend/src/composables/editor/useSidebarPrefs.ts
import { ref, watch } from 'vue'

export const SIDEBAR_PREFS_KEY = 'yhfish.editor.sidebar'

export interface SidebarPrefs {
  leftSidebarCollapsed: boolean
  inspectorCollapsed: boolean
  /** 'palette' = 节点库 (创建模式); 'snippets' = 用户资产 (复用模式). */
  leftSidebarTab: 'palette' | 'snippets'
  varsExpanded: boolean
  snapEnabled: boolean
}

const DEFAULTS: SidebarPrefs = {
  leftSidebarCollapsed: false,
  inspectorCollapsed: false,
  leftSidebarTab: 'palette',
  varsExpanded: true,
  snapEnabled: true,
}

function loadInitial(): SidebarPrefs {
  const merged: SidebarPrefs = { ...DEFAULTS }
  try {
    const raw = localStorage.getItem(SIDEBAR_PREFS_KEY)
    if (raw) Object.assign(merged, JSON.parse(raw))
  } catch (_e) {
    // localStorage 不可用或 JSON 坏 → 保留 defaults
  }
  return merged
}

// 模块级单例 — 所有 caller 共享一份 prefs + 一个 watch.
// 跟 useSidebarCollapsed.ts 同款单例模式 (避免 per-call orphan watch + 状态不同步).
const prefs = ref<SidebarPrefs>(loadInitial())

// deep required: callers 直接 mutate 单字段 (prefs.value.varsExpanded = false), 而非整 obj 重赋.
const stopWatch = watch(
  prefs,
  (p) => {
    try {
      localStorage.setItem(SIDEBAR_PREFS_KEY, JSON.stringify(p))
    } catch (_e) {
      // localStorage 配额满 / 不可用 → 静默 (prefs 仍在内存工作)
    }
  },
  { deep: true },
)

// HMR: module reload 时 dispose 旧 watcher, 防 zombie 继续写 localStorage.
if (import.meta.hot) {
  import.meta.hot.dispose(() => {
    stopWatch()
  })
}

export function useSidebarPrefs() {
  return { prefs }
}
