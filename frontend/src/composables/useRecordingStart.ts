import { onScopeDispose, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { backend } from '@/lib/backend'
import { useHotkeysStore } from '@/stores/hotkeys'
import { useRecordingStore, type RecordingInvocation, type RecordingMode } from '@/stores/recording'

export function useRecordingStart() {
  const recording = useRecordingStore()
  const hotkeys = useHotkeysStore()
  const countdown = ref(0)
  const starting = ref(false)
  let generation = 0

  async function start(
    mode: RecordingMode,
    targetSlot: string,
    origin: RecordingInvocation,
  ): Promise<boolean> {
    if (starting.value || recording.state.phase !== 'idle') return false
    const current = ++generation
    starting.value = true
    try {
      await backend.recording.validateTarget(targetSlot)
      try {
        await backend.tools.openRecordingHUD()
      } catch (error) {
        console.warn('recording HUD unavailable', error)
      }
      await hotkeys.reload().catch(() => undefined)
      const stopKey =
        hotkeys.list.find((entry) => entry.key === 'recording.stop')?.hotkeyStr || 'F12'
      const pauseKey =
        hotkeys.list.find((entry) => entry.key === 'recording.pause')?.hotkeyStr || 'F11'
      for (let seconds = 3; seconds >= 1; seconds--) {
        if (current !== generation) return false
        countdown.value = seconds
        Events.Emit('recording:countdown', { sec: seconds, mode, stopKey, pauseKey })
        await new Promise((resolve) => setTimeout(resolve, 1000))
      }
      if (current !== generation) return false
      countdown.value = 0
      await recording.start(mode, targetSlot, origin)
      return true
    } catch (error) {
      await backend.tools.closeRecordingHUD().catch(() => undefined)
      throw error
    } finally {
      if (current === generation) {
        countdown.value = 0
        starting.value = false
      }
    }
  }

  function cancelCountdown(): void {
    if (countdown.value <= 0) return
    generation++
    countdown.value = 0
    starting.value = false
    Events.Emit('recording:countdown', { sec: 0 })
    void backend.tools.closeRecordingHUD().catch(() => undefined)
  }

  onScopeDispose(cancelCountdown)

  return { countdown, starting, start, cancelCountdown }
}
