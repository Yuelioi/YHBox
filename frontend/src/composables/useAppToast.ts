import { useToast as useNuxtToast } from '@nuxt/ui/composables'

type ToastInput = Parameters<ReturnType<typeof useNuxtToast>['add']>[0]

const TOAST_ICONS: Partial<Record<NonNullable<ToastInput['color']>, string>> = {
  error: 'i-tabler-alert-circle',
  success: 'i-tabler-circle-check',
  warning: 'i-tabler-alert-triangle',
  info: 'i-tabler-info-circle',
}

export function normalizeAppToast(toast: ToastInput): ToastInput {
  const error = toast.color === 'error'
  return {
    ...toast,
    icon: toast.icon ?? (toast.color ? TOAST_ICONS[toast.color] : undefined),
    progress: false,
    close: toast.close ?? error,
    duration: error ? 0 : toast.duration,
  }
}

export function useToast() {
  const toast = useNuxtToast()
  return {
    ...toast,
    add(input: ToastInput) {
      return toast.add(normalizeAppToast(input))
    },
    update(id: string | number, input: Partial<ToastInput>) {
      return toast.update(id, normalizeAppToast(input as ToastInput))
    },
  }
}
