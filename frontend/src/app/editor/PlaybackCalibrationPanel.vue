<template>
  <section
    v-if="clipBlob"
    data-testid="playback-calibration"
    class="rounded-lg border border-default bg-elevated/30 p-3"
  >
    <div class="flex items-center gap-2">
      <UIcon name="i-tabler-rotate-360" class="size-4 text-primary" />
      <h3 class="text-xs font-semibold text-highlighted">
        {{ t('workflow.inspector.playback_calibration') }}
      </h3>
      <UBadge v-if="loading" color="neutral" variant="soft" size="sm">
        {{ t('common.loading') }}
      </UBadge>
    </div>

    <div class="mt-2 grid grid-cols-2 gap-2">
      <div v-for="item in values" :key="item.label" class="min-w-0 rounded-md bg-default px-2 py-2">
        <p class="truncate text-[10px] text-dimmed">{{ item.label }}</p>
        <p class="mt-0.5 truncate font-mono text-xs font-semibold text-toned">{{ item.value }}</p>
        <p v-if="item.hint" class="mt-0.5 truncate text-[9px] text-dimmed">{{ item.hint }}</p>
      </div>
    </div>

    <p v-if="failure" class="mt-2 text-[10px] text-warning">{{ failure }}</p>
    <p v-else class="mt-2 text-[10px] leading-4 text-muted">
      {{ t('workflow.inspector.playback_calibration_formula') }}
    </p>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Node } from './EditorSession'
import { resolvePlaybackCalibration } from './playbackCalibration'
import { backend, type InputClipSummary } from '@/lib/backend'
import { useAssetsStore } from '@/stores/assets'
import { useSettingsStore } from '@/stores/settings'

const props = defineProps<{
  node: Node
  targetSlot: string
}>()
const { t } = useI18n()
const assets = useAssetsStore()
const settings = useSettingsStore()
const summary = ref<InputClipSummary | null>(null)
const loading = ref(false)
const failure = ref('')
let generation = 0

const clipBlob = computed(() => {
  const binding = props.node.bindings.clip
  return binding?.kind === 'blob' ? binding.blob : undefined
})
const target = computed(() =>
  settings.data?.automation.targets.find((candidate) => candidate.slot === props.targetSlot),
)
const calibration = computed(() =>
  resolvePlaybackCalibration(
    summary.value?.meta.mouseCounts360 ?? 0,
    target.value,
    settings.activeMouseCounts360,
  ),
)
const values = computed(() => [
  {
    label: t('workflow.inspector.playback_source_counts'),
    value: calibration.value.sourceCounts > 0 ? `${calibration.value.sourceCounts}` : '—',
    hint: summary.value ? t('workflow.inspector.playback_source_recorded') : '',
  },
  {
    label: t('workflow.inspector.playback_target_counts'),
    value: calibration.value.targetCounts > 0 ? `${calibration.value.targetCounts}` : '—',
    hint: t(`workflow.inspector.playback_target_${calibration.value.targetSource}`),
  },
])

watch(
  () => [clipBlob.value?.mediaType, clipBlob.value?.digest, clipBlob.value?.size, assets.epoch],
  () => void loadSummary(),
  { immediate: true },
)
onMounted(() => {
  if (!settings.loaded) void settings.load()
})

async function loadSummary(): Promise<void> {
  const current = ++generation
  summary.value = null
  failure.value = ''
  const blob = clipBlob.value
  if (!blob) return
  loading.value = true
  try {
    const binding = await assets.resolveBinding(blob)
    if (current !== generation) return
    if (!binding.found || binding.kind !== 'clip') throw new Error('clip asset is unavailable')
    const clip = await backend.clips.summary(binding.guid)
    if (current === generation) summary.value = clip
  } catch {
    if (current === generation)
      failure.value = t('workflow.inspector.playback_metadata_unavailable')
  } finally {
    if (current === generation) loading.value = false
  }
}
</script>
