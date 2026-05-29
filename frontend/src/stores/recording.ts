// recording store — 后端录制状态机的【纯镜像】.
//
// 单一真相源是后端 recording.Service (Phase: idle|recording|finalizing). 本 store 不自己存
// 可 desync 的 isRecording flag — state 只由两个来源更新:
//   ① 'recording:state' 事件 (后端每次转换广播全量 state)
//   ② reconcile() 主动调 GetState() 对账 (窗口聚焦 / 编辑器挂载时, 丢事件自愈)
//
// 命令 (start/stop) 幂等: 后端不在录时 stop 返 null 不报错; 本地不乐观置态, 一切以后端为准.
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { Events } from '@wailsio/runtime'
import { backend } from '@/lib/backend'
import { i18n } from '@/i18n'

export interface RecordingState {
  phase: 'idle' | 'recording' | 'finalizing'
  containerID: string
  filterMode: string
  tempID: string
  startedAtMs: number
}

export interface RecordingStopPayload {
  subgraphID: string
  containerID: string
  label: string
  filterMode: string
}

const IDLE: RecordingState = { phase: 'idle', containerID: '', filterMode: '', tempID: '', startedAtMs: 0 }

function normalize(st: any): RecordingState {
  return {
    phase: st?.phase === 'recording' || st?.phase === 'finalizing' ? st.phase : 'idle',
    containerID: st?.containerID ?? '',
    filterMode: st?.filterMode ?? '',
    tempID: st?.tempID ?? '',
    startedAtMs: st?.startedAtMs ?? 0,
  }
}

export const useRecordingStore = defineStore('recording', () => {
  const state = ref<RecordingState>({ ...IDLE })
  const lastResult = ref<RecordingStopPayload | null>(null)

  // 派生值 — 无独立 flag, 不可能跟后端 desync.
  const isRecording = computed(() => state.value.phase === 'recording')
  // 录制目标容器 (A2 删除守卫 / A3 离开确认 / A4 指示器 读它). idle 时为空.
  const activeTargetContainerID = computed(() =>
    state.value.phase === 'idle' ? '' : state.value.containerID,
  )

  function applyState(st: any) {
    state.value = normalize(st)
  }

  // 后端权威状态广播 — 镜像入口. store 实例化时注册一次 (每窗口一份, 长生命周期不退订).
  Events.On('recording:state', (e: any) => {
    const st = e?.data?.[0] ?? e?.data ?? e
    if (st && typeof st === 'object') applyState(st)
  })

  // reconcile 主动跟后端对账. 窗口聚焦 / 编辑器挂载时调 — 任何 recording:state 事件丢失,
  // 下次对账自动收敛, 卡死状态自愈.
  async function reconcile() {
    try {
      const st = await backend.recording.getState()
      if (st) applyState(st)
    } catch (e) {
      console.warn('recording reconcile failed', e)
    }
  }

  async function start(filterMode: 'precise' | 'simple', containerID: string): Promise<void> {
    if (isRecording.value) return
    if (!containerID) throw new Error('recording.start: containerID ' + i18n.global.t('common.required'))
    lastResult.value = null
    await backend.recording.start({ filterMode, containerID })
    // 不乐观置态 — 后端 Start 成功即广播 recording:state(recording); 这里对账一次兜底.
    await reconcile()
  }

  async function stop(): Promise<RecordingStopPayload | null> {
    // 幂等: 后端不在录 → 返 null 不抛错. 拿到产物 (或 null) 后对账收敛状态.
    const payload = (await backend.recording.stop()) as RecordingStopPayload | null | undefined
    lastResult.value = payload ?? null
    await reconcile()
    return payload ?? null
  }

  return { state, lastResult, isRecording, activeTargetContainerID, applyState, reconcile, start, stop }
})
