// useScreenPick NodeInspector "屏幕拾取" 按钮的逻辑层. 打开独立 ScreenPicker 子窗口,
// 用户在窗口内点选 / 框选 → wails event 'tools:picker-result' → 回填 node.config 字段.
//
// 哪些 kind 能 pickPoint / pickRect: 由 canPick* 计算属性根据当前 node.kind 决定.
import { computed, ref, type Ref } from 'vue'
import { backend, type GraphNode } from '@/lib/backend'
import { awaitWailsEvent } from '@/composables/useWailsEvent'

type PointPayload = { xRatio: number; yRatio: number; cancelled?: boolean }
type RectPayload = { region: [number, number, number, number]; cancelled?: boolean }

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
  /** 接收 picker 结果时调用, 通常是 NodeInspector 的 setCfg / setCfgBatch */
  applyPoint: (xRatio: number, yRatio: number) => void
  applyRect: (region: [number, number, number, number]) => void
}) {
  const picking = ref(false)

  function getNode(): GraphNode | null {
    return typeof opts.node === 'function' ? opts.node() : opts.node.value
  }

  const canPickPoint = computed(() => {
    const k = getNode()?.kind ?? ''
    return k === 'ClickAt' || k === 'Scroll'
  })
  const canPickRect = computed(() => {
    const k = getNode()?.kind ?? ''
    return k === 'DetectColor'
  })

  async function openPicker<T>(mode: 'point' | 'rect'): Promise<T | null> {
    const id = genID()
    picking.value = true
    try {
      // 先挂监听再开窗口, 防 race
      const waiter = awaitWailsEvent<PickerResult<T>>('tools:picker-result', (p) => p?.id === id)
      const r = await backend.tools.openScreenPicker(mode, id)
      if (r === undefined) return null
      const result = await waiter
      const cancelled = (result.payload as { cancelled?: boolean } | undefined)?.cancelled
      if (cancelled) return null
      return result.payload
    } finally {
      picking.value = false
    }
  }

  async function onPickPoint() {
    const p = await openPicker<PointPayload>('point')
    if (p) opts.applyPoint(p.xRatio, p.yRatio)
  }

  async function onPickRect() {
    const p = await openPicker<RectPayload>('rect')
    if (p) opts.applyRect(p.region)
  }

  async function onOpenHUD() {
    await backend.tools.openMouseHUD()
  }

  return { picking, canPickPoint, canPickRect, onPickPoint, onPickRect, onOpenHUD }
}
