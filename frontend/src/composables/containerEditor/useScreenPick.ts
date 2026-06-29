// useScreenPick NodeInspector "屏幕拾取" 按钮的逻辑层. 打开独立 ScreenPicker 子窗口,
// 用户在窗口内点选 / 框选 → wails event 'tools:picker-result' → 回填 node.config 字段.
//
// onPickRect 已泛化为 fieldPath 模型 — GeometryWidget 等可按需调用.
import { computed, ref, type Ref } from 'vue'
import { backend, type GraphNode } from '@/lib/backend'
import { awaitWailsEvent } from '@/composables/useWailsEvent'
import { useTemplatesStore } from '@/stores/templates'

export type RectPayload = {
  region: [number, number, number, number]
  screenW?: number
  screenH?: number
  cancelled?: boolean
}
export type ColorPayload = { range: number[]; hueWrap: boolean; cancelled?: boolean }

interface PickerResult<T = unknown> {
  id: string
  mode: string
  payload: T
}

function genID(): string {
  return 'pick-' + Math.random().toString(36).slice(2, 10) + '-' + Date.now()
}

export function useScreenPick(opts: {
  node: Ref<GraphNode | null> | (() => GraphNode | null)
  /**
   * 接收 rect 结果时调用.
   * @param fieldPath 调用方透传的字段路径 (供回填定位用); 也可忽略直接写固定字段.
   * @param region    ratio [x,y,w,h] 0-1
   */
  applyRect: (fieldPath: string, region: [number, number, number, number]) => void
  /**
   * 接收颜色范围结果时调用.
   * @param fieldPath 字段路径 (同 applyRect)
   * @param range     颜色范围数组
   * @param hueWrap   色相是否跨 0/360 回绕
   */
  applyColor: (fieldPath: string, range: number[], hueWrap: boolean) => void
}) {
  const picking = ref(false)
  const tplStore = useTemplatesStore()

  function getNode(): GraphNode | null {
    return typeof opts.node === 'function' ? opts.node() : opts.node.value
  }

  const canPickRect = computed(() => {
    const k = getNode()?.kind ?? ''
    return k === 'DetectColor'
  })

  async function openPicker<T>(mode: 'point' | 'rect' | 'color', colorSpace = ''): Promise<T | null> {
    if (picking.value) return null // 防并发: 已有拾取进行中, 忽略重复触发 (吸管按钮无 :loading 守卫, 在此兜底)
    const id = genID()
    picking.value = true
    try {
      // 先挂监听再开窗口, 防 race
      const waiter = awaitWailsEvent<PickerResult<T>>('tools:picker-result', (p) => p?.id === id)
      const r = await backend.tools.openScreenPicker(mode, id, tplStore.containerId, getNode()?.id ?? '', colorSpace)
      if (r === undefined) return null
      const result = await waiter
      const cancelled = (result.payload as { cancelled?: boolean } | undefined)?.cancelled
      if (cancelled) return null
      return result.payload
    } finally {
      picking.value = false
    }
  }

  /**
   * 打开 rect 截图框选; 结果通过 applyRect(fieldPath, region) 回填.
   * GeometryWidget 等可直接调用; NodeInspector 的旧 Region 字段也走这里.
   */
  async function onPickRect(fieldPath: string = 'Region') {
    const p = await openPicker<RectPayload>('rect')
    if (p) opts.applyRect(fieldPath, p.region)
  }

  async function onPickColor(fieldPath: string, colorSpace: 'hsv' | 'rgb') {
    const p = await openPicker<ColorPayload>('color', colorSpace)
    if (p) opts.applyColor(fieldPath, p.range, p.hueWrap)
  }

  async function onOpenHUD() {
    await backend.tools.openMouseHUD(tplStore.containerId)
  }

  return { picking, canPickRect, onPickRect, onPickColor, onOpenHUD }
}
