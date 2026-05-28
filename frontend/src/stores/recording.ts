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
      lastResult.value = null
    } catch (e) {
      console.error('recording.start failed', e)
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
    }
  }

  return { isRecording, tempID, lastResult, start, stop }
})
