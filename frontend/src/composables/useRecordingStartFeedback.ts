import { useToast } from '@nuxt/ui/composables'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { errorMessage, normalizeError } from '@/lib/invoke'

export function useRecordingStartFeedback() {
  const toast = useToast()
  const router = useRouter()
  const { t } = useI18n()

  function show(title: string, error: unknown): void {
    const normalized = normalizeError(error)
    const calibrationRequired =
      normalized.code === 'RECORDING_CALIBRATION_REQUIRED' ||
      normalized.message?.trim() === 'RECORDING_CALIBRATION_REQUIRED'
    toast.add({
      title,
      description: errorMessage(error),
      color: 'error',
      actions: calibrationRequired
        ? [
            {
              label: t('workflow.recording.open_calibration'),
              icon: 'i-tabler-target-arrow',
              color: 'neutral',
              variant: 'soft',
              onClick: () => {
                void router.push({ path: '/settings', query: { section: 'input' } })
              },
            },
          ]
        : undefined,
    })
  }

  return { show }
}
