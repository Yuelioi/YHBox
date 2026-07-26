import { backend } from '@/lib/backend'
import { awaitWailsEvent } from '@/composables/useWailsEvent'

export type TargetPoint = {
  x: number
  y: number
  xRatio: number
  yRatio: number
  screenW: number
  screenH: number
  cancelled?: boolean
}

export type TargetRegion = {
  x: number
  y: number
  w: number
  h: number
  region: [number, number, number, number]
  screenW: number
  screenH: number
  cancelled?: boolean
}

export type TargetColorRange = {
  range: [number, number, number, number, number, number]
  hueWrap: boolean
  cancelled?: boolean
}

export type TargetDimensions = { width: number; height: number }

export async function targetDimensions(targetSlot: string): Promise<TargetDimensions> {
  if (!targetSlot) throw new Error('an automation target is required for coordinate conversion')
  const result = (await backend.tools.mousePos(targetSlot)) as unknown as {
    clientW?: number
    clientH?: number
  }
  const width = Number(result?.clientW)
  const height = Number(result?.clientH)
  if (!(width > 0) || !(height > 0)) {
    throw new Error('the target client area is unavailable')
  }
  return { width, height }
}

export async function pickTargetValue<T extends TargetPoint | TargetRegion | TargetColorRange>(
  mode: 'point' | 'rect' | 'color',
  targetSlot: string,
  colorSpace = '',
): Promise<T | null> {
  if (!targetSlot) return null
  const requestID = `authoring-${mode}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
  const controller = new AbortController()
  const waiter = awaitWailsEvent<{ id: string; payload?: T }>(
    'tools:picker-result',
    (result) => result?.id === requestID,
    controller.signal,
  )
  try {
    await backend.tools.openScreenPicker(mode, requestID, targetSlot, colorSpace)
  } catch (error) {
    controller.abort(error)
    await waiter.catch(() => undefined)
    throw error
  }
  const result = await waiter
  return result.payload && !result.payload.cancelled ? result.payload : null
}
