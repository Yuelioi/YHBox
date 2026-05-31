import { defineStore } from 'pinia'
import { ref } from 'vue'
import { backend } from '@/lib/backend'

// 跟 Go services.Settings 对齐（v2：fish/cook/piano/battle 等 v1 游戏专属字段已删）
export interface Settings {
  ui: {
    logger: {
      panelOpen: boolean
      autoScroll: boolean
      showTime: boolean
      showTag: boolean
      wrapText: boolean
      writeFile: boolean
      fileDir: string
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
    mouseCounts360: number // 鼠标转 360° 累积 HID counts；0 = 未校准
  }
  locale: 'zh' | 'en' // i18n 口子；目前仅 zh 有翻译
  capture: {
    method: 'auto' | 'gdi' | 'wgc' | 'mock' // 截屏后端；改完需重启 exe 才生效
    dumpDebug: boolean // bot detect 落盘带框 PNG 到 debug/captures/
  }
}

export const useSettingsStore = defineStore('settings', () => {
  const data = ref<Settings | null>(null)
  const loaded = ref(false)

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

  return { data, loaded, load, patch }
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
