// recording store — 当前录制状态 + 最近一次 stop 拿到的 payload.
// v2 Subgraph-only: Start({filterMode, containerID}) → tempID,
// Stop() → {subgraphID, containerID, label, filterMode}.
// 异步停录 (F12 / HUD): 走 'recording:completed' event, 不经 stop().
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { backend } from '@/lib/backend'
import { i18n } from '@/i18n'

export interface RecordingStopPayload {
  subgraphID: string
  containerID: string
  label: string
  filterMode: string
}

export const useRecordingStore = defineStore('recording', () => {
  const isRecording = ref(false)
  const tempID = ref<string>('')
  const lastResult = ref<RecordingStopPayload | null>(null)
  // A1: 录制目标容器的单一来源. 录制开始时锁定 (= 子图最终落盘的容器), 任何收尾路径清空.
  // 全 FE "正在录哪个容器" 只认这个值: A2 删除守卫 / A3 离开确认 / A4 指示器 都读它.
  // F12/HUD 异步停录走 'recording:completed' (不经 stop()), 那条路径在 useRecording 里清.
  const activeTargetContainerID = ref<string>('')

  async function start(
    filterMode: 'precise' | 'simple',
    containerID: string,
  ): Promise<void> {
    if (isRecording.value) return
    if (!containerID) throw new Error('recording.start: containerID ' + i18n.global.t('common.required'))
    try {
      const id = (await backend.recording.start({ filterMode, containerID })) as string | undefined
      tempID.value = id ?? ''
      isRecording.value = true
      activeTargetContainerID.value = containerID
      lastResult.value = null
    } catch (e) {
      console.error('recording.start failed', e)
      activeTargetContainerID.value = ''
      throw e
    }
  }

  async function stop(): Promise<RecordingStopPayload | null> {
    try {
      const payload = (await backend.recording.stop()) as RecordingStopPayload | null | undefined
      lastResult.value = payload ?? null
      isRecording.value = false
      return payload ?? null
    } catch (e) {
      console.error('recording.stop failed', e)
      isRecording.value = false
      throw e
    } finally {
      activeTargetContainerID.value = ''
    }
  }

  // markStopped: 异步停录路径 (F12/HUD 的 'recording:completed' 事件) 复位状态. 不经 stop() RPC.
  function markStopped() {
    isRecording.value = false
    activeTargetContainerID.value = ''
  }

  return { isRecording, tempID, lastResult, activeTargetContainerID, start, stop, markStopped }
})
