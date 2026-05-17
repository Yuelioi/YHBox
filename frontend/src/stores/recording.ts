// recording store — 当前录制状态 + 最近一次 stop 拿到的 clip.
// 新 RPC (Phase 1+2): Start({filterMode}) → tempID, Stop() → InputClip.
// 异步停录 (F12 / HUD): 走 'recording:completed' event, 不经 stop().
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { backend } from '@/lib/backend'
import type { InputClip } from '@/stores/clips'

export const useRecordingStore = defineStore('recording', () => {
  const isRecording = ref(false)
  const tempID = ref<string>('')
  const lastClip = ref<InputClip | null>(null)

  async function start(filterMode: 'precise' | 'simple' = 'precise'): Promise<void> {
    if (isRecording.value) return
    try {
      const id = (await backend.recording.start({ filterMode })) as string | undefined
      tempID.value = id ?? ''
      isRecording.value = true
      lastClip.value = null
    } catch (e) {
      console.error('recording.start failed', e)
      throw e
    }
  }

  async function stop(): Promise<InputClip | null> {
    try {
      const clip = (await backend.recording.stop()) as InputClip | null | undefined
      lastClip.value = clip ?? null
      isRecording.value = false
      return clip ?? null
    } catch (e) {
      console.error('recording.stop failed', e)
      isRecording.value = false
      throw e
    }
  }

  return { isRecording, tempID, lastClip, start, stop }
})
