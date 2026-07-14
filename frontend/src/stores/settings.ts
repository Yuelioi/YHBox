import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { backend, type AIConnection } from '@/lib/backend'

// MouseProfile 命名鼠标校准档（跟 Go services.MouseProfile 对齐）。
// counts360 = 原地转 360° 鼠标硬件累积 |dx|；同机不同游戏内灵敏度不同 → 多 profile。
export interface MouseProfile {
  label: string
  counts360: number
}

// LauncherBlock 悬浮窗启动器的一个块（积木式编排，跟 Go services.LauncherBlock 对齐）。
// type: 'container'(容器按钮) | 'label'(文字标题) | 'hsep'(水平分隔符) | 'vsep'(垂直分隔符)。
export interface LauncherBlock {
  id: string
  type: 'container' | 'label' | 'hsep' | 'vsep'
  containerId?: string // type=container
  icon?: string // type=container 自定义图标（完整 tabler 名）
  label?: string // container 自定义名 / label 标题文字
}

// 跟 Go services.Settings 对齐（v2：fish/cook/piano/battle 等 v1 游戏专属字段已删）
export interface Settings {
  ui: {
    logger: {
      enabled: boolean
      liveView: boolean
      level: 'debug' | 'info' | 'warn' | 'error'
      panelOpen: boolean
      autoScroll: boolean
      showTime: boolean
      showTag: boolean
      wrapText: boolean
      writeFile: boolean
      fileDir: string
      showNodeEnter?: boolean
    }
    window: {
      width: number
      height: number
    }
    autostart: boolean // 开机自启（写 HKCU Run 注册表）
    minimizeToTray: boolean // 关闭按钮 → 隐藏到托盘
    actionStopHotkey: string // 全局强停热键（默认 "Ctrl+Shift+F9"），改完即时生效（热键中心 rebind）
    calibrateHotkey: string // DPI 校准启动/停止热键（默认 "F8"），改完即时生效
    recordingStopHotkey: string // 录制停止热键（默认 "F12"，LL hook 拦截不透传游戏）
    recordingPauseHotkey: string // 录制暂停/继续切换热键（默认 "F11"）
    recordingMouseMode: 'relative' | 'absolute' // 录制鼠标语义；改完需重启
    mouseProfiles: MouseProfile[] // 命名鼠标校准档列表（异环/原神…各一档）
    activeMouseProfile: string // 指向 mouseProfiles 里某个 label；空/失配 → activeMouseCounts360 兜底
    launcherItems: LauncherBlock[] // 悬浮窗启动器块序列（设置里编排，有序，积木式）
    launcherDisplay: string // 'both'(默认)|'icon'|'text'
    launcherToggleHotkey: string // 呼出/隐藏悬浮窗的全局热键（空=未绑）
  }
  locale: 'zh' | 'en' // i18n 口子
  capture: {
    method: 'auto' | 'gdi' | 'wgc' | 'mock' // 截屏后端；改完需重启 exe 才生效
    dumpDebug: boolean // bot detect 落盘带框 PNG 到 debug/captures/
  }
  ai: {
    connections: AIConnection[]
    default: string // 指向某 connection.id；新建 AI 节点缺省连接
  }
  mcp: {
    armed: boolean // MCP 服务是否启用；Go Armed → TS armed
  }
}

export const useSettingsStore = defineStore('settings', () => {
  const data = ref<Settings | null>(null)
  const loaded = ref(false)

  // activeMouseProfile 命中 → 该档 counts；失配但只有一档 → 那一档；否则 0。
  // 跟 Go Settings.ActiveMouseCounts360() 逻辑一致。
  const mouseProfiles = computed<MouseProfile[]>(() => data.value?.ui.mouseProfiles ?? [])
  const activeMouseCounts360 = computed<number>(() => {
    const list = mouseProfiles.value
    const active = data.value?.ui.activeMouseProfile
    const hit = list.find((p) => p.label === active)
    if (hit) return hit.counts360
    if (list.length === 1) return list[0].counts360
    return 0
  })

  async function load() {
    const s = await backend.settings.get()
    if (s) {
      data.value = s as Settings
      loaded.value = true
    }
  }

  // patch 是 partial Settings（RFC7386 deep merge 语义）
  async function patch(p: object) {
    const result = await backend.settings.update(p)
    if (result === undefined) return // 失败已 toast
    // Go 端不发 settings:changed 事件，前端本地立刻 deep-merge 进 store，
    // 让任何在监听 data 的 view 立即看到新值，不必等下一次 load。
    if (data.value) {
      deepMerge(data.value, p)
    }
  }

  // patchAIConnections 整替 connections（RFC7386 数组整替）；删档清 default 时一并传 def=''。
  async function patchAIConnections(connections: AIConnection[], def?: string) {
    const ai: { connections: AIConnection[]; default?: string } = { connections }
    if (def !== undefined) ai.default = def
    await patch({ ai })
  }

  return { data, loaded, load, patch, patchAIConnections, mouseProfiles, activeMouseCounts360 }
})

function deepMerge(target: any, source: any) {
  for (const k of Object.keys(source)) {
    if (source[k] !== null && typeof source[k] === 'object' && !Array.isArray(source[k])) {
      if (!target[k]) target[k] = {}
      deepMerge(target[k], source[k])
    } else {
      target[k] = source[k]
    }
  }
}
