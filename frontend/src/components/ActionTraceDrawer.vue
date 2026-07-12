<template>
  <BaseModal
    v-model:open="modelOpen"
    :title="t('log.action_trace.title')"
    icon="i-tabler-route"
    icon-color="info"
    size="4xl"
    tall
  >
    <template #header-extra>
      <span class="text-[11px] text-dimmed tabular-nums">
        {{ t('log.action_trace.count', { n: traces.length }) }}
      </span>
    </template>

    <div v-if="rows.length === 0" class="py-10 text-center text-xs text-dimmed italic">
      {{ t('log.action_trace.empty') }}
    </div>

    <div v-else class="space-y-1.5 font-mono text-[11px]">
      <div
        v-for="row in rows"
        :key="row.key"
        class="grid grid-cols-[84px_minmax(0,1fr)_auto] gap-3 border-b border-default/70 px-1.5 py-2 last:border-b-0"
      >
        <div class="space-y-1 text-dimmed">
          <div class="tabular-nums">{{ formatTime(row.trace.startedAt || row.trace.endedAt) }}</div>
          <div class="flex items-center gap-1.5">
            <span class="size-1.5 rounded-full" :class="statusDotClass(row.trace.status)" />
            <span class="uppercase" :class="statusTextClass(row.trace.status)">
              {{ row.trace.status || 'unknown' }}
            </span>
          </div>
        </div>

        <div class="min-w-0 space-y-1">
          <div class="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
            <span class="text-highlighted font-semibold">{{ row.trace.action }}</span>
            <span class="text-sky-300">{{ sourceLabel(row.trace) }}</span>
            <span v-if="targetLabel(row.trace)" class="text-dimmed">@</span>
            <span v-if="targetLabel(row.trace)" class="text-emerald-300 truncate">
              {{ targetLabel(row.trace) }}
            </span>
          </div>

          <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-dimmed">
            <span v-if="row.trace.backend"
              >{{ t('log.action_trace.backend') }}={{ row.trace.backend }}</span
            >
            <span v-if="hasDuration(row.trace)"
              >{{ t('log.action_trace.duration') }}={{ row.trace.durationMs }}ms</span
            >
            <span v-if="stepCount(row.trace)"
              >{{ t('log.action_trace.coords') }}={{ stepCount(row.trace) }}</span
            >
            <span v-if="row.trace.containerId">container={{ row.trace.containerId }}</span>
          </div>

          <div v-if="row.trace.error" class="text-error break-all">
            {{ t('log.action_trace.error') }}: {{ row.trace.error }}
          </div>

          <details v-if="row.request || row.result" class="group">
            <summary class="cursor-pointer select-none text-dimmed hover:text-toned">
              {{ t('log.action_trace.payload') }}
            </summary>
            <pre
              class="mt-1 max-h-40 overflow-auto rounded bg-sunken px-2 py-1 text-[10px] leading-snug text-toned"
              >{{ payloadText(row) }}</pre
            >
          </details>
        </div>

        <div class="text-right text-dimmed tabular-nums">#{{ row.number }}</div>
      </div>
    </div>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseModal from '@/components/common/BaseModal.vue'
import { useLogStore, type ActionTraceEntry } from '@/stores/log'

const props = defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  'update:open': [v: boolean]
}>()

const { t } = useI18n()
const logStore = useLogStore()

const modelOpen = computed({
  get: () => props.open,
  set: (v: boolean) => emit('update:open', v),
})

const traces = computed(() => logStore.actionTraces)
const rows = computed(() =>
  traces.value
    .map((trace, idx) => ({
      trace,
      key: `${trace.startedAt || trace.endedAt || idx}:${idx}`,
      number: idx + 1,
      request: trace.request,
      result: trace.result,
    }))
    .reverse(),
)

function sourceLabel(trace: ActionTraceEntry) {
  const source = trace.source ?? {}
  const nodeKind = source.nodeKind ?? source.NodeKind ?? '?'
  const nodeId = source.nodeId ?? source.NodeID ?? '?'
  const inPin = source.inPin ?? source.InPin ?? ''
  return `${nodeKind}(${nodeId})${inPin ? '.' + inPin : ''}`
}

function targetLabel(trace: ActionTraceEntry) {
  return trace.target?.id ?? trace.target?.ID ?? ''
}

function stepCount(trace: ActionTraceEntry) {
  return trace.coordinateSteps?.length ?? 0
}

function hasDuration(trace: ActionTraceEntry) {
  return Number.isFinite(trace.durationMs)
}

function formatTime(iso?: string) {
  if (!iso) return '--:--:--.---'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '--:--:--.---'
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  const ss = String(d.getSeconds()).padStart(2, '0')
  const ms = String(d.getMilliseconds()).padStart(3, '0')
  return `${hh}:${mm}:${ss}.${ms}`
}

function statusDotClass(status?: string) {
  switch (status) {
    case 'ok':
    case 'success':
      return 'bg-success'
    case 'error':
    case 'failed':
      return 'bg-error'
    default:
      return 'bg-warning'
  }
}

function statusTextClass(status?: string) {
  switch (status) {
    case 'ok':
    case 'success':
      return 'text-success'
    case 'error':
    case 'failed':
      return 'text-error'
    default:
      return 'text-warning'
  }
}

function payloadText(row: { request: unknown; result: unknown }) {
  try {
    return JSON.stringify(
      {
        request: row.request,
        result: row.result,
      },
      null,
      2,
    )
  } catch {
    return String(row.request ?? row.result ?? '')
  }
}
</script>
