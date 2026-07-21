import { ref } from 'vue'
import { backend } from '@/lib/backend'
import { useRecordingStore, type RecordingInvocation, type RecordingMode } from '@/stores/recording'

export function useRecordingStart() {
  const recording = useRecordingStore()
  const starting = ref(false)

  async function start(
    mode: RecordingMode,
    targetSlot: string,
    origin: RecordingInvocation,
  ): Promise<boolean> {
    if (starting.value || recording.state.phase !== 'idle') return false
    starting.value = true
    try {
      await backend.recording.validateTarget(targetSlot)
      try {
        await backend.tools.openRecordingHUD()
      } catch (error) {
        console.warn('recording HUD unavailable', error)
      }
      await recording.start(mode, targetSlot, origin)
      return true
    } catch (error) {
      await backend.tools.closeRecordingHUD().catch(() => undefined)
      throw error
    } finally {
      starting.value = false
    }
  }

  return { starting, start }
}
