<template>
  <section
    class="flex shrink-0 flex-col bg-default"
    :class="embedded ? 'h-full max-h-none border-0' : 'max-h-64 border-t border-default'"
  >
    <header class="flex items-center gap-3 border-b border-default px-4 py-2.5">
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <span class="size-2 rounded-full" :class="statusColor" aria-hidden="true" />
          <h2 class="text-xs font-semibold text-highlighted">
            {{ t('workflow.timeline.run_status', { status: run.status }) }}
          </h2>
        </div>
        <p class="mt-0.5 truncate font-mono text-[10px] text-dimmed">{{ run.runId }}</p>
      </div>
      <UButton
        v-if="canCancel"
        :label="t('workflow.action.stop')"
        icon="i-tabler-square"
        color="error"
        variant="soft"
        size="xs"
        @click="emit('cancel')"
      />
      <UButton
        :label="t('workflow.timeline.export')"
        icon="i-tabler-download"
        color="neutral"
        variant="ghost"
        size="xs"
        :loading="exporting"
        @click="emit('export')"
      />
      <UButton
        :label="t('workflow.action.refresh')"
        icon="i-tabler-refresh"
        color="neutral"
        variant="ghost"
        size="xs"
        @click="emit('refresh')"
      />
      <UButton
        v-if="!embedded"
        icon="i-tabler-x"
        color="neutral"
        variant="ghost"
        size="xs"
        :aria-label="t('workflow.timeline.close')"
        @click="emit('close')"
      />
    </header>

    <div class="min-h-0 flex-1 overflow-y-auto px-4 py-3">
      <UButton
        v-if="activeAttempt"
        data-testid="run-active-attempt"
        color="neutral"
        variant="soft"
        class="mb-3 grid h-auto w-full grid-cols-[minmax(0,1fr)_auto] items-start justify-stretch gap-3 border border-primary/25 px-3 py-2.5 text-left"
        @click="emit('focus-node', activeAttempt.graphPath, activeAttempt.nodeId)"
      >
        <span class="min-w-0">
          <span class="flex items-center gap-2 text-xs font-medium text-highlighted">
            <span class="size-2 animate-pulse rounded-full bg-primary motion-reduce:animate-none" />
            {{ t('workflow.timeline.active_attempt') }} ·
            {{ nodeLabels?.[activeAttempt.nodeId] || activeAttempt.nodeId }}
          </span>
          <span class="mt-1 block truncate font-mono text-[10px] text-muted">
            {{ activeAttemptStatus }}
          </span>
        </span>
        <span class="text-right font-mono text-[10px] text-dimmed">
          <span class="block">{{ activeAttemptElapsed }}</span>
          <span v-if="activeAttemptTimeout" class="mt-1 block">
            {{ t('workflow.timeline.timeout_budget', { value: activeAttemptTimeout }) }}
          </span>
        </span>
      </UButton>
      <div v-if="run.failure" class="mb-3 rounded-lg border border-error/35 bg-error/10 px-3 py-2">
        <p class="text-xs font-medium text-error">{{ failureMessage }}</p>
        <p class="mt-1 text-[11px] text-muted">
          {{ run.failure.category }}{{ run.failure.nodeId ? ` / ${run.failure.nodeId}` : '' }}
        </p>
      </div>
      <div v-if="run.timelineTotal > run.timeline.length" class="mb-3 flex items-center gap-2">
        <span class="mr-auto text-[11px] text-muted">
          {{
            t('workflow.timeline.page', {
              page: run.timelinePage,
              pages: run.timelinePages,
              total: run.timelineTotal,
            })
          }}
        </span>
        <UButton
          size="xs"
          color="neutral"
          variant="ghost"
          icon="i-tabler-chevron-left"
          :disabled="run.timelinePage >= run.timelinePages"
          :label="t('workflow.timeline.older')"
          @click="emit('page', run.timelinePage + 1)"
        />
        <UButton
          size="xs"
          color="neutral"
          variant="ghost"
          trailing-icon="i-tabler-chevron-right"
          :disabled="run.timelinePage <= 1"
          :label="t('workflow.timeline.newer')"
          @click="emit('page', run.timelinePage - 1)"
        />
      </div>
      <p v-if="!run.timeline.length" class="py-3 text-center text-xs text-muted">
        {{ t('workflow.timeline.empty') }}
      </p>
      <ol v-else class="space-y-2">
        <li v-for="entry in run.timeline" :key="entry.sequence" class="rounded-lg bg-elevated/45">
          <UButton
            color="neutral"
            variant="ghost"
            class="grid h-auto w-full grid-cols-[72px_minmax(0,1fr)_auto] items-start justify-stretch gap-3 px-3 py-2 text-left"
            :disabled="!entry.nodeId"
            @click="entry.nodeId && emit('focus-node', entry.graphPath, entry.nodeId)"
          >
            <span class="font-mono text-[10px] text-dimmed"
              >#{{ entry.sequence }} {{ entry.kind }}</span
            >
            <span class="min-w-0">
              <span class="block truncate text-xs text-toned">{{
                entry.nodeId || entry.summary.code
              }}</span>
              <span
                v-if="entry.attemptOutcome || entry.action || entry.statusCode || entry.errorCode"
                class="mt-0.5 block truncate font-mono text-[10px] text-muted"
              >
                {{ entry.attemptOutcome || entry.action || entry.statusCode || entry.errorCode }}
              </span>
              <span
                v-if="isUnhandledRoute(entry)"
                class="mt-1 block text-[10px] font-medium text-warning"
              >
                {{ t('workflow.timeline.unhandled_route', { route: 'timeout' }) }}
              </span>
            </span>
            <span class="text-right font-mono text-[10px]">
              <span
                class="block text-toned"
                :aria-label="
                  t('workflow.timeline.since_start', {
                    value: formatTimelineOffset(entry.occurredAt, run.queuedAt),
                  })
                "
              >
                {{ formatTimelineOffset(entry.occurredAt, run.queuedAt) }}
              </span>
              <time
                class="mt-0.5 block text-dimmed"
                :datetime="entry.occurredAt"
                :title="formatTimelineDateTime(entry.occurredAt)"
                :aria-label="
                  t('workflow.timeline.occurred_at', {
                    value: formatTimelineDateTime(entry.occurredAt),
                  })
                "
              >
                {{ formatTimelineClock(entry.occurredAt) }}
              </time>
              <span v-if="entry.attempt > 0" class="mt-0.5 block text-dimmed">
                {{ t('workflow.timeline.attempt', { n: entry.attempt }) }}
              </span>
            </span>
          </UButton>
        </li>
      </ol>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useNow } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import type { RunView } from '@/app/transport/workflow'
import { activeRunAttempt, runRouteKey, statusRoutePort } from './runTrace'
import { formatTimelineClock, formatTimelineDateTime, formatTimelineOffset } from './timelineTime'

const props = defineProps<{
  run: RunView
  embedded?: boolean
  nodeLabels?: Record<string, string>
  unhandledRoutes?: string[]
  exporting?: boolean
}>()
const emit = defineEmits<{
  cancel: []
  refresh: []
  close: []
  'focus-node': [graphPath: string[], nodeId: string]
  page: [page: number]
  export: []
}>()
const { t, te } = useI18n()
const now = useNow({ interval: 1000 })
const unhandledRouteSet = computed(() => new Set(props.unhandledRoutes ?? []))
const activeAttempt = computed(() => activeRunAttempt(props.run))
const activeAttemptElapsed = computed(() => {
  const startedAt = Date.parse(activeAttempt.value?.startedAt ?? '')
  if (!Number.isFinite(startedAt)) return '—'
  return formatElapsed(now.value.getTime() - startedAt)
})
const activeAttemptTimeout = computed(() => {
  const timeout = activeAttempt.value?.counters.timeout_ms
  return typeof timeout === 'number' && timeout > 0 ? formatElapsed(timeout) : ''
})
const activeAttemptStatus = computed(() => {
  const code = activeAttempt.value?.statusCode
  if (!code) return t('workflow.timeline.executing')
  const key = `workflow.timeline.status.${code}`
  return te(key) ? t(key) : code
})

const canCancel = computed(() => ['QUEUED', 'RUNNING'].includes(props.run.status.toUpperCase()))
const failureMessage = computed(() => {
  if (!props.run.failure) return ''
  const key = `error.${props.run.failure.code}`
  return te(key) ? t(key) : props.run.failure.code
})
const statusColor = computed(() => {
  switch (props.run.status.toUpperCase()) {
    case 'SUCCEEDED':
      return 'bg-success'
    case 'FAILED':
    case 'INTERRUPTED':
      return 'bg-error'
    case 'CANCELLED':
      return 'bg-warning'
    default:
      return 'bg-primary animate-pulse motion-reduce:animate-none'
  }
})

function formatElapsed(milliseconds: number): string {
  const seconds = Math.max(0, Math.floor(milliseconds / 1000))
  if (seconds < 60) return `${seconds}s`
  return `${Math.floor(seconds / 60)}m ${seconds % 60}s`
}

function isUnhandledRoute(entry: RunView['timeline'][number]): boolean {
  const port = statusRoutePort(entry.statusCode)
  return Boolean(
    port &&
    entry.nodeId &&
    unhandledRouteSet.value.has(runRouteKey(entry.graphPath, entry.nodeId, port)),
  )
}
</script>
