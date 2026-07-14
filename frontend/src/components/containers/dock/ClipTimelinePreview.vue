<template>
  <div
    class="relative flex h-full w-full flex-col justify-between overflow-hidden bg-sunken px-4 py-3"
    role="img"
    :aria-label="t('assetBrowser.clipPreview', { duration, count: eventCount })"
  >
    <div class="flex items-center justify-between text-xs text-muted">
      <span>{{ mouseModeLabel }}</span>
      <span>{{ resolution }}</span>
    </div>
    <div class="relative">
      <div class="h-px bg-accented" />
      <div class="absolute -top-1 left-0 size-2 rounded-full bg-primary" />
      <div class="absolute -top-1 right-0 size-2 rounded-full border border-primary bg-default" />
      <div class="absolute inset-x-0 -top-5 text-center text-sm font-semibold text-highlighted">
        {{ duration }}
      </div>
    </div>
    <div class="flex items-center justify-between text-xs text-dimmed">
      <span>0</span>
      <span>{{ t('assetBrowser.inputEvents', { n: eventCount }) }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const {
  durationUs,
  eventCount,
  mouseMode = '',
  baseResolution = [0, 0],
} = defineProps<{
  durationUs: number
  eventCount: number
  mouseMode?: string
  baseResolution?: [number, number]
}>()
const { t } = useI18n()

const duration = computed(() => {
  const ms = durationUs / 1000
  return ms < 1000 ? `${Math.round(ms)} ms` : `${(ms / 1000).toFixed(1)} s`
})
const resolution = computed(() =>
  baseResolution[0] > 0
    ? `${baseResolution[0]}×${baseResolution[1]}`
    : t('assetBrowser.noResolution'),
)
const mouseModeLabel = computed(() => {
  const mode = ['relative', 'absolute', 'mixed'].includes(mouseMode) ? mouseMode : 'unknown'
  return t(`assetBrowser.mouseMode.${mode}`)
})
</script>
