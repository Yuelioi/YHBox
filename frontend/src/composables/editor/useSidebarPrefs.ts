// frontend/src/composables/editor/useSidebarPrefs.ts
import { ref, watch } from 'vue'

export const SIDEBAR_PREFS_KEY = 'yotta.editor.sidebar'

export type EditorExperienceMode = 'basic' | 'pro'

export interface SidebarPrefs {
  /** 基础模式收起变量、调试和技术元数据；专业模式暴露完整编辑能力。 */
  experienceMode: EditorExperienceMode
  /** 左侧 VS Code 活动栏式: 哪个停靠面板打开 (null = 只剩细图标栏, 画布最大). */
  leftDrawer: 'vars' | 'snippets' | 'nodes' | 'assets' | null
  /** 资产停靠面板当前 tab (模板 / 子图库 / Clip / 资源管理). */
  assetTab: 'templates' | 'library' | 'clips' | 'maintenance'
  inspectorCollapsed: boolean
  varsExpanded: boolean
  snapEnabled: boolean
  /** 连线渲染样式 (vue-flow edge type): default=贝塞尔曲线 / smoothstep=圆角直角 / step=折线. */
  edgeStyle: 'default' | 'smoothstep' | 'step'
}

const DEFAULTS: SidebarPrefs = {
  experienceMode: 'basic',
  leftDrawer: null,
  assetTab: 'templates',
  inspectorCollapsed: false,
  varsExpanded: true,
  snapEnabled: true,
  edgeStyle: 'default',
}

function loadInitial(): SidebarPrefs {
  const merged: SidebarPrefs = { ...DEFAULTS }
  try {
    const raw = localStorage.getItem(SIDEBAR_PREFS_KEY)
    if (raw) {
      const saved = JSON.parse(raw) as Partial<SidebarPrefs>
      Object.assign(merged, saved)
      // 升级前没有模式字段的既有用户继续看到完整工具，避免功能突然消失。
      if (saved.experienceMode !== 'basic' && saved.experienceMode !== 'pro') {
        merged.experienceMode = 'pro'
      }
    }
  } catch {
    // localStorage 不可用或 JSON 坏 → 保留 defaults
  }
  return merged
}

// 模块级单例 — 所有 caller 共享一份 prefs + 一个 watch (避免 per-call orphan watch + 状态不同步).
const prefs = ref<SidebarPrefs>(loadInitial())

// deep required: callers 直接 mutate 单字段 (prefs.value.varsExpanded = false), 而非整 obj 重赋.
const stopWatch = watch(
  prefs,
  (p) => {
    try {
      localStorage.setItem(SIDEBAR_PREFS_KEY, JSON.stringify(p))
    } catch {
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
