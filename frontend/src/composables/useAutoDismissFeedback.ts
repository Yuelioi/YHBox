import { watch, type Ref } from 'vue'

export const SUCCESS_FEEDBACK_DURATION_MS = 4_000

export function useAutoDismissFeedback<T extends { tone: string }>(feedback: Ref<T | null>): void {
  watch(
    feedback,
    (current, _previous, onCleanup) => {
      if (current?.tone !== 'success') return

      const snapshot = current
      const timer = window.setTimeout(() => {
        if (feedback.value === snapshot) feedback.value = null
      }, SUCCESS_FEEDBACK_DURATION_MS)
      onCleanup(() => window.clearTimeout(timer))
    },
    { flush: 'sync' },
  )
}
