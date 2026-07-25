<template>
  <section class="overflow-hidden rounded-xl border border-default bg-default">
    <header class="flex flex-wrap items-center gap-3 border-b border-default px-3 py-2.5">
      <div class="min-w-52 flex-1">
        <h3 class="text-sm font-semibold text-highlighted">{{ t('preciseWorkbench.title') }}</h3>
        <p class="mt-0.5 text-xs text-muted">
          {{
            t('preciseWorkbench.summary', {
              count: preview.eventCount,
              duration: formatTime(durationUs),
            })
          }}
        </p>
      </div>
      <UBadge color="primary" variant="soft" size="sm">
        {{ t('preciseWorkbench.source_counts_360') }}:
        {{ environment.mouseCounts360 || '—' }}
      </UBadge>
      <UButton
        size="sm"
        color="neutral"
        variant="ghost"
        :icon="detailsOpen ? 'i-tabler-chevron-up' : 'i-tabler-adjustments-horizontal'"
        :label="t('preciseWorkbench.recording_details')"
        :aria-expanded="detailsOpen"
        @click="toggleDetails"
      />
    </header>

    <div
      v-if="calibrationRisk"
      class="flex items-start gap-2 border-b border-warning/30 bg-warning/10 px-3 py-2 text-xs text-warning"
    >
      <UIcon name="i-tabler-alert-triangle" class="mt-0.5 size-4 shrink-0" />
      <span>{{ t('preciseWorkbench.calibration_warning') }}</span>
    </div>

    <div class="space-y-3 border-b border-default p-3">
      <div>
        <h4 class="text-xs font-medium text-toned">{{ t('preciseWorkbench.timeline_title') }}</h4>
        <p class="mt-0.5 text-[11px] text-muted">{{ t('preciseWorkbench.timeline_hint') }}</p>
      </div>
      <button
        data-recording-trim-timeline
        type="button"
        class="relative h-8 w-full cursor-crosshair overflow-hidden rounded-md bg-elevated outline-none ring-primary transition-shadow focus-visible:ring-2"
        :aria-label="t('preciseWorkbench.timeline_aria')"
        @click="chooseTrimBoundary"
      >
        <span
          class="absolute inset-y-0 bg-primary/20"
          :style="{ left: percent(trimStartUs), right: `${100 - numericPercent(trimEndUs)}%` }"
        />
        <span
          class="absolute inset-y-0 w-1 -translate-x-1/2 rounded bg-primary"
          :style="{ left: percent(trimStartUs) }"
          aria-hidden="true"
        />
        <span
          class="absolute inset-y-0 w-1 -translate-x-1/2 rounded bg-primary"
          :style="{ left: percent(trimEndUs) }"
          aria-hidden="true"
        />
      </button>
      <div
        v-for="track in preview.tracks"
        :key="track.kind"
        class="grid grid-cols-[7rem_minmax(0,1fr)_3rem] items-center gap-2"
      >
        <span class="truncate text-xs font-medium text-toned">{{ trackLabel(track.kind) }}</span>
        <div class="relative h-2 overflow-hidden rounded-full bg-elevated">
          <div
            class="absolute inset-y-0 rounded-full bg-primary/70"
            :style="trackStyle(track.firstUs, track.lastUs)"
          />
        </div>
        <span class="text-right font-mono text-[10px] text-dimmed">{{ track.count }}</span>
      </div>
      <p v-if="!preview.tracks.length" class="py-4 text-center text-xs text-muted">
        {{ t('preciseWorkbench.no_tracks') }}
      </p>
    </div>

    <div v-if="editableTrim" class="grid gap-3 border-b border-default p-3 md:grid-cols-2">
      <UFormField
        :label="t('preciseWorkbench.trim_start')"
        :hint="t('preciseWorkbench.milliseconds')"
      >
        <UInputNumber
          :model-value="Math.round(trimStartUs / 1000)"
          :min="0"
          :max="Math.max(0, Math.round((trimEndUs - 1) / 1000))"
          :step="10"
          class="w-full"
          @update:model-value="setTrimStart"
        />
      </UFormField>
      <UFormField
        :label="t('preciseWorkbench.trim_end')"
        :hint="t('preciseWorkbench.milliseconds')"
      >
        <UInputNumber
          :model-value="Math.round(trimEndUs / 1000)"
          :min="Math.round((trimStartUs + 1) / 1000)"
          :max="Math.round(durationUs / 1000)"
          :step="10"
          class="w-full"
          @update:model-value="setTrimEnd"
        />
      </UFormField>
      <p class="md:col-span-2 text-xs text-muted">{{ t('preciseWorkbench.trim_hint') }}</p>
    </div>

    <div v-if="detailsOpen" class="border-b border-default bg-elevated/10">
      <div class="grid gap-2 border-b border-default p-3 sm:grid-cols-2 lg:grid-cols-4">
        <InfoTile icon="i-tabler-dimensions" :label="t('preciseWorkbench.resolution')">
          {{ environment.baseResolution[0] }}×{{ environment.baseResolution[1] }}
        </InfoTile>
        <InfoTile icon="i-tabler-mouse" :label="t('preciseWorkbench.mouse_mode')">
          {{ environment.mouseMode || '—' }}
        </InfoTile>
        <InfoTile icon="i-tabler-rotate-360" :label="t('preciseWorkbench.counts_360')">
          {{ environment.mouseCounts360 || '—' }}
        </InfoTile>
        <InfoTile icon="i-tabler-stack-2" :label="t('preciseWorkbench.track_count')">
          {{ preview.tracks.length }}
        </InfoTile>
      </div>
      <div class="flex items-center gap-2 border-b border-default px-3 py-2">
        <div class="mr-auto min-w-0">
          <p class="text-xs font-medium text-toned">{{ t('preciseWorkbench.raw_events') }}</p>
          <p class="mt-0.5 text-[10px] text-dimmed">{{ t('preciseWorkbench.raw_events_hint') }}</p>
        </div>
        <span class="text-[10px] text-dimmed">
          {{ rawPage.offset + (rawPage.items.length ? 1 : 0) }}–{{
            Math.min(rawPage.offset + rawPage.items.length, rawPage.total)
          }}
          / {{ rawPage.total }}
        </span>
        <UButton
          icon="i-tabler-chevron-left"
          size="xs"
          color="neutral"
          variant="ghost"
          :aria-label="t('preciseWorkbench.raw_previous')"
          :disabled="rawPage.offset <= 0 || rawLoading"
          @click="loadRaw(Math.max(0, rawPage.offset - rawPage.limit))"
        />
        <UButton
          icon="i-tabler-chevron-right"
          size="xs"
          color="neutral"
          variant="ghost"
          :aria-label="t('preciseWorkbench.raw_next')"
          :disabled="rawPage.offset + rawPage.items.length >= rawPage.total || rawLoading"
          @click="loadRaw(rawPage.offset + rawPage.limit)"
        />
      </div>
      <div class="max-h-48 overflow-auto font-mono text-[10px]">
        <div
          v-for="event in rawPage.items"
          :key="`${event.tUs}:${event.seq}`"
          data-recording-event-row
          class="grid grid-cols-[6rem_7rem_1fr] gap-3 border-b border-default/50 px-3 py-1.5 text-muted"
        >
          <span>{{ formatTime(event.tUs) }}</span>
          <span class="text-toned">{{ eventTypeLabel(event.type) }}</span>
          <span>a={{ event.a }} · b={{ event.b }} · c={{ event.c }} · seq={{ event.seq }}</span>
        </div>
        <p v-if="rawLoading" class="px-3 py-6 text-center text-muted">{{ t('common.loading') }}</p>
        <p v-else-if="rawFailure" class="px-3 py-4 text-error" role="alert">
          {{ rawFailure }}
        </p>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend, type InputEventPage } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'
import type { RecordingEnvironment, RecordingPreview } from '@/stores/recording'
import InfoTile from '@/components/common/InfoTile.vue'
import type { WorkflowResource } from '../../../../contracts/workflow/current/workflow-source'

const props = defineProps<{
  preview: RecordingPreview
  environment: RecordingEnvironment
  durationUs: number
  trimStartUs: number
  trimEndUs: number
  pendingId?: string
  clipId?: string
  workflowResource?: WorkflowResource
  editableTrim?: boolean
}>()
const emit = defineEmits<{
  'update:trimStartUs': [value: number]
  'update:trimEndUs': [value: number]
}>()
const { t } = useI18n()
const detailsOpen = ref(false)
const rawLoading = ref(false)
const rawFailure = ref('')
const rawPage = ref<InputEventPage>({ items: [], total: 0, offset: 0, limit: 100 })

const calibrationRisk = computed(
  () =>
    (props.environment.mouseMode === 'relative' || props.environment.mouseMode === 'mixed') &&
    props.environment.mouseCounts360 <= 0,
)

function trackLabel(kind: RecordingPreview['tracks'][number]['kind']): string {
  return t(`preciseWorkbench.track_${kind.replaceAll('-', '_')}`)
}

function trackStyle(firstUs: number, lastUs: number): Record<string, string> {
  const left = numericPercent(firstUs)
  const width = Math.max(0.5, numericPercent(lastUs) - left)
  return { left: `${left}%`, width: `${width}%` }
}

function numericPercent(value: number): number {
  if (props.durationUs <= 0) return 0
  return Math.max(0, Math.min(100, (value / props.durationUs) * 100))
}

function percent(value: number): string {
  return `${numericPercent(value)}%`
}

function chooseTrimBoundary(event: MouseEvent): void {
  if (!props.editableTrim || props.durationUs <= 0) return
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  if (rect.width <= 0) return
  const value = Math.round(
    Math.max(0, Math.min(1, (event.clientX - rect.left) / rect.width)) * props.durationUs,
  )
  if (Math.abs(value - props.trimStartUs) <= Math.abs(value - props.trimEndUs)) {
    emit('update:trimStartUs', Math.min(value, props.trimEndUs - 1))
  } else {
    emit('update:trimEndUs', Math.max(value, props.trimStartUs + 1))
  }
}

function setTrimStart(value: unknown): void {
  const next = Math.max(0, Math.min(props.trimEndUs - 1, Math.round((Number(value) || 0) * 1000)))
  emit('update:trimStartUs', next)
}

function setTrimEnd(value: unknown): void {
  const next = Math.max(
    props.trimStartUs + 1,
    Math.min(props.durationUs, Math.round((Number(value) || 0) * 1000)),
  )
  emit('update:trimEndUs', next)
}

async function toggleDetails(): Promise<void> {
  detailsOpen.value = !detailsOpen.value
  if (detailsOpen.value && !rawPage.value.items.length) await loadRaw(0)
}

async function loadRaw(offset: number): Promise<void> {
  if (!props.pendingId && !props.clipId && !props.workflowResource) return
  rawLoading.value = true
  rawFailure.value = ''
  try {
    if (props.pendingId) {
      rawPage.value = await backend.recording.pendingEvents(
        props.pendingId,
        offset,
        rawPage.value.limit,
      )
    } else if (props.workflowResource) {
      rawPage.value = await backend.workflowResources.events(
        props.workflowResource,
        offset,
        rawPage.value.limit,
      )
    } else {
      rawPage.value = await backend.clips.events(props.clipId!, offset, rawPage.value.limit)
    }
  } catch (error) {
    rawFailure.value = errorMessage(error)
  } finally {
    rawLoading.value = false
  }
}

function eventTypeLabel(type: number): string {
  return t(
    `preciseWorkbench.event_${['none', 'key_down', 'key_up', 'mouse_down', 'mouse_up', 'move', 'raw_delta', 'scroll'][type] ?? 'unknown'}`,
  )
}

function formatTime(microseconds: number): string {
  return `${(Math.max(0, microseconds) / 1_000_000).toFixed(3)}s`
}
</script>
