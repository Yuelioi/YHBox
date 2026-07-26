<template>
  <section
    class="flex shrink-0 flex-col bg-default"
    :class="embedded ? 'h-full max-h-none border-0' : 'max-h-96 border-t border-default'"
    data-testid="workflow-debugger"
  >
    <header class="flex items-center gap-3 border-b border-default px-4 py-3">
      <div
        class="flex size-8 shrink-0 items-center justify-center rounded-md"
        :class="statusSurface"
        aria-hidden="true"
      >
        <UIcon :name="statusIcon" class="size-4" />
      </div>
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <h2 class="text-xs font-semibold text-highlighted">{{ t('workflow.debug.title') }}</h2>
          <UBadge :color="statusColor" variant="soft" size="xs" data-testid="debug-status">
            {{ t(`workflow.debug.status_${snapshot.status.replace('-', '_')}`) }}
          </UBadge>
        </div>
        <p class="mt-0.5 truncate text-xs text-muted" data-testid="debug-status-message">
          {{ statusMessage }}
        </p>
      </div>

      <div class="flex shrink-0 items-center gap-1.5">
        <UButton
          v-if="paused"
          data-testid="workflow-debug-step"
          :label="t('workflow.debug.step')"
          icon="i-tabler-player-track-next"
          size="sm"
          :loading="busy"
          :disabled="busy"
          @click="emit('step')"
        />
        <UButton
          v-if="paused"
          data-testid="workflow-debug-continue"
          :label="t('workflow.debug.continue')"
          icon="i-tabler-player-play"
          color="neutral"
          variant="soft"
          size="sm"
          :disabled="busy"
          @click="emit('continue')"
        />
        <UButton
          v-else-if="!completed"
          data-testid="workflow-debug-pause"
          :label="t('workflow.debug.pause')"
          icon="i-tabler-player-pause"
          color="neutral"
          variant="soft"
          size="sm"
          :disabled="busy || snapshot.status === 'pause-pending'"
          @click="emit('pause')"
        />
        <UButton
          v-if="!completed"
          data-testid="workflow-debug-stop"
          :label="t('workflow.action.stop')"
          icon="i-tabler-square"
          color="error"
          variant="soft"
          size="sm"
          :disabled="busy"
          @click="emit('stop')"
        />
        <UButton
          v-if="!embedded"
          icon="i-tabler-x"
          color="neutral"
          variant="ghost"
          size="sm"
          :aria-label="t('workflow.debug.close')"
          @click="emit('close')"
        />
      </div>
    </header>

    <div class="min-h-0 flex-1 overflow-y-auto px-4 py-3">
      <div v-if="hasExecutionPosition" class="flex items-stretch gap-2">
        <UButton
          v-if="snapshot.previousNodeId"
          color="neutral"
          variant="ghost"
          class="h-auto min-w-0 flex-1 justify-start rounded-md border border-default px-3 py-2 text-left"
          @click="focusPrevious"
        >
          <span class="min-w-0">
            <span class="block text-[10px] font-medium text-dimmed">
              {{ t('workflow.debug.just_executed') }}
            </span>
            <span class="mt-0.5 block truncate text-xs text-toned">
              {{ nodeTitle(snapshot.previousNodeId) }}
            </span>
          </span>
        </UButton>
        <UIcon
          v-if="snapshot.previousNodeId && snapshot.nodeId"
          name="i-tabler-arrow-right"
          class="size-4 shrink-0 self-center text-dimmed"
        />
        <UButton
          v-if="snapshot.nodeId"
          color="neutral"
          :variant="paused ? 'soft' : 'ghost'"
          class="h-auto min-w-0 flex-1 justify-start rounded-md border px-3 py-2 text-left"
          :class="paused ? 'border-warning/50' : 'border-default'"
          data-testid="debug-current-node"
          @click="focusCurrent"
        >
          <span class="min-w-0">
            <span
              class="block text-[10px] font-medium"
              :class="paused ? 'text-warning' : 'text-dimmed'"
            >
              {{ currentPositionLabel }}
            </span>
            <span class="mt-0.5 block truncate text-xs text-highlighted">
              {{ nodeTitle(snapshot.nodeId) }}
            </span>
          </span>
        </UButton>
        <UIcon
          v-if="snapshot.nodeId && nextEntry && !completed"
          name="i-tabler-arrow-right"
          class="size-4 shrink-0 self-center text-dimmed"
        />
        <UButton
          v-if="nextEntry && !completed"
          color="neutral"
          variant="ghost"
          class="h-auto min-w-0 flex-1 justify-start rounded-md border border-default px-3 py-2 text-left"
          @click="focusNext"
        >
          <span class="min-w-0">
            <span class="block text-[10px] font-medium text-dimmed">
              {{ t('workflow.debug.next_queued') }}
            </span>
            <span class="mt-0.5 block truncate text-xs text-toned">
              {{ nodeTitle(nextEntry?.nodeId) }}
            </span>
          </span>
        </UButton>
      </div>
      <p v-else class="text-xs text-muted">{{ t('workflow.debug.waiting') }}</p>

      <div class="mt-3 border-t border-default pt-3">
        <div v-if="detailSections.length" class="flex min-h-0 flex-col gap-2">
          <nav
            class="flex flex-wrap items-center gap-1"
            :aria-label="t('workflow.debug.runtime_data')"
          >
            <UButton
              v-for="section in detailSections"
              :key="section.id"
              size="xs"
              color="neutral"
              :variant="activeSectionId === section.id ? 'soft' : 'ghost'"
              @click="activeSectionId = section.id"
            >
              {{ section.title }}
              <UBadge color="neutral" variant="soft" size="xs">{{ section.count }}</UBadge>
            </UButton>
          </nav>

          <div
            v-if="activeSection?.kind === 'facts'"
            class="grid gap-2 md:grid-cols-2 xl:grid-cols-3"
          >
            <div
              v-for="fact in activeSection.facts"
              :key="fact.key"
              class="min-w-0 rounded-md border border-default bg-elevated/25 px-3 py-2"
            >
              <p class="truncate text-[11px] font-medium text-toned">{{ fact.key }}</p>
              <p class="mt-0.5 truncate font-mono text-[10px] text-dimmed">
                {{ valueSummary(fact.value) }}
              </p>
            </div>
          </div>
          <ol
            v-else-if="activeSection?.kind === 'queue'"
            class="grid gap-2 md:grid-cols-2 xl:grid-cols-3"
          >
            <li
              v-for="(entry, index) in activeSection.entries"
              :key="`${entry.graphPath?.join('/') || entry.graphId}:${entry.nodeId}:${index}`"
            >
              <UButton
                color="neutral"
                variant="ghost"
                class="h-auto w-full justify-start rounded-md border border-default px-3 py-2 text-left"
                @click="focusQueueEntry(entry)"
              >
                <span class="min-w-0">
                  <span class="block text-[10px] text-dimmed">#{{ index + 1 }}</span>
                  <span class="block truncate text-xs text-toned">{{
                    nodeTitle(entry.nodeId)
                  }}</span>
                </span>
              </UButton>
            </li>
          </ol>
        </div>
        <p v-else class="text-xs text-muted">{{ t('workflow.debug.no_runtime_data') }}</p>

        <details class="mt-2 text-[10px] text-dimmed">
          <summary class="w-fit cursor-pointer select-none hover:text-muted">
            {{ t('workflow.debug.technical_details') }}
          </summary>
          <p class="mt-1 font-mono">
            {{ snapshot.graphPath?.join(' / ') || snapshot.graphId || '—' }} · gen
            {{ snapshot.generation
            }}<template v-if="snapshot.runStatus"> · {{ snapshot.runStatus }}</template>
          </p>
        </details>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DebugSnapshot } from '@/app/transport/workflow'
import type {
  DebugQueueEntry,
  DebugValueView,
} from '@bindings/github.com/yottaapp/yotta/internal/workflow/compiler/models.js'

interface DebugFact {
  key: string
  value: DebugValueView
}

type DetailSection =
  | { id: string; title: string; count: number; kind: 'facts'; facts: DebugFact[] }
  | { id: string; title: string; count: number; kind: 'queue'; entries: DebugQueueEntry[] }

const props = defineProps<{
  snapshot: DebugSnapshot
  embedded?: boolean
  busy?: boolean
  nodeLabels?: Record<string, string>
}>()
const emit = defineEmits<{
  continue: []
  pause: []
  step: []
  stop: []
  close: []
  'focus-node': [graphPath: string[], nodeId: string]
}>()
const { t } = useI18n()
const activeSectionId = ref('inputs')

const paused = computed(() => props.snapshot.status === 'paused')
const completed = computed(() => props.snapshot.status === 'completed')
const nextEntry = computed(() => props.snapshot.queue[0])
const hasExecutionPosition = computed(() =>
  Boolean(
    props.snapshot.previousNodeId || props.snapshot.nodeId || (nextEntry.value && !completed.value),
  ),
)
const statusColor = computed<'neutral' | 'warning' | 'primary' | 'error'>(() => {
  if (completed.value && props.snapshot.runStatus === 'FAILED') return 'error'
  if (completed.value) return 'neutral'
  if (paused.value || props.snapshot.status === 'pause-pending') return 'warning'
  return 'primary'
})
const statusIcon = computed(() => {
  if (completed.value)
    return props.snapshot.runStatus === 'FAILED' ? 'i-tabler-alert-triangle' : 'i-tabler-flag'
  if (paused.value) return 'i-tabler-player-pause-filled'
  if (props.snapshot.status === 'pause-pending') return 'i-tabler-hourglass'
  return 'i-tabler-player-play-filled'
})
const statusSurface = computed(() => {
  if (statusColor.value === 'error') return 'bg-error/10 text-error'
  if (statusColor.value === 'warning') return 'bg-warning/10 text-warning'
  if (statusColor.value === 'primary') return 'bg-primary/10 text-primary'
  return 'bg-elevated text-muted'
})
const statusMessage = computed(() => {
  const node = nodeTitle(props.snapshot.nodeId)
  if (paused.value) return t('workflow.debug.paused_before', { node })
  if (props.snapshot.status === 'pause-pending') return t('workflow.debug.pause_pending_hint')
  if (completed.value) {
    if (props.snapshot.runStatus === 'FAILED') return t('workflow.debug.finished_failed')
    if (props.snapshot.runStatus === 'CANCELLED') return t('workflow.debug.finished_cancelled')
    return t('workflow.debug.finished')
  }
  return t('workflow.debug.running_hint')
})
const currentPositionLabel = computed(() => {
  if (paused.value) return t('workflow.debug.will_execute')
  if (completed.value) return t('workflow.debug.end_position')
  return t('workflow.debug.current_position')
})
const stateFacts = computed<DebugFact[]>(() =>
  Object.entries(props.snapshot.state).flatMap(([name, state]) =>
    state ? [{ key: name, value: state.value }] : [],
  ),
)
const inputFacts = computed<DebugFact[]>(() =>
  Object.entries(props.snapshot.inputs).flatMap(([key, value]) => (value ? [{ key, value }] : [])),
)
const outputFacts = computed<DebugFact[]>(() =>
  Object.entries(props.snapshot.outputs).flatMap(([nodeId, outputs]) =>
    Object.entries(outputs ?? {}).flatMap(([portId, value]) =>
      value ? [{ key: `${nodeTitle(nodeId)} · ${portId}`, value }] : [],
    ),
  ),
)
const detailSections = computed<DetailSection[]>(() => {
  const sections: DetailSection[] = []
  if (inputFacts.value.length) {
    sections.push({
      id: 'inputs',
      title: t('workflow.debug.inputs'),
      count: inputFacts.value.length,
      kind: 'facts',
      facts: inputFacts.value,
    })
  }
  if (outputFacts.value.length) {
    sections.push({
      id: 'outputs',
      title: t('workflow.debug.outputs'),
      count: outputFacts.value.length,
      kind: 'facts',
      facts: outputFacts.value,
    })
  }
  if (stateFacts.value.length) {
    sections.push({
      id: 'state',
      title: t('workflow.debug.state'),
      count: stateFacts.value.length,
      kind: 'facts',
      facts: stateFacts.value,
    })
  }
  if (props.snapshot.queue.length) {
    sections.push({
      id: 'queue',
      title: t('workflow.debug.queue'),
      count: props.snapshot.queue.length,
      kind: 'queue',
      entries: props.snapshot.queue,
    })
  }
  return sections
})
const activeSection = computed(
  () =>
    detailSections.value.find((section) => section.id === activeSectionId.value) ??
    detailSections.value[0],
)

watch(
  detailSections,
  (sections) => {
    if (!sections.some((section) => section.id === activeSectionId.value)) {
      activeSectionId.value = sections[0]?.id ?? 'inputs'
    }
  },
  { immediate: true },
)

function nodeTitle(nodeId?: string): string {
  if (!nodeId) return t('workflow.debug.waiting')
  return props.nodeLabels?.[nodeId] ?? nodeId
}

function valueSummary(value: DebugValueView): string {
  const identity = value.digest ? value.digest.slice(0, 18) : t('workflow.debug.redacted')
  return `${value.representation} · ${value.size} B · ${identity}`
}

function focusCurrent(): void {
  if (!props.snapshot.nodeId) return
  focusNode(
    props.snapshot.graphPath ?? (props.snapshot.graphId ? [props.snapshot.graphId] : []),
    props.snapshot.nodeId,
  )
}

function focusPrevious(): void {
  if (!props.snapshot.previousNodeId) return
  focusNode(
    props.snapshot.previousGraphPath ??
      (props.snapshot.previousGraphId ? [props.snapshot.previousGraphId] : []),
    props.snapshot.previousNodeId,
  )
}

function focusNext(): void {
  if (nextEntry.value) focusQueueEntry(nextEntry.value)
}

function focusQueueEntry(entry: DebugQueueEntry): void {
  focusNode(entry.graphPath ?? (entry.graphId ? [entry.graphId] : []), entry.nodeId)
}

function focusNode(graphPath: string[], nodeId: string): void {
  emit('focus-node', graphPath, nodeId)
}
</script>
