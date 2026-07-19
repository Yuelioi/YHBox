<template>
  <article
    class="workflow-node relative min-w-[230px] overflow-visible rounded-lg border bg-elevated shadow-sm transition-[border-color,box-shadow] duration-150"
    :class="visualState.surfaceClasses"
  >
    <span
      v-if="visualState.executionTone"
      data-testid="node-execution-stripe"
      class="pointer-events-none absolute inset-y-2 left-0 w-0.5 rounded-r-full"
      :class="visualState.executionStripeClasses"
      aria-hidden="true"
    />
    <header
      class="workflow-node-drag-handle flex cursor-grab items-center gap-2 rounded-t-lg border-b border-default bg-muted/35 px-3 py-2.5 active:cursor-grabbing"
    >
      <UIcon :name="iconName" class="size-4 shrink-0 text-primary" aria-hidden="true" />
      <div class="min-w-0 flex-1">
        <p class="truncate text-xs font-semibold text-highlighted">{{ title }}</p>
        <p class="truncate font-mono text-[10px] text-dimmed">{{ projection.execution.class }}</p>
      </div>
      <UIcon
        v-if="node.disabled"
        name="i-tabler-ban"
        class="size-3.5 text-warning"
        :aria-label="t('workflow.node.disabled')"
      />
      <UIcon
        v-if="visualState.diagnosticTone"
        data-testid="node-diagnostic-status"
        :name="diagnosticIcon"
        class="size-3.5"
        :class="diagnosticClass"
        :aria-label="t(`workflow.diagnostics.${visualState.diagnosticTone}`)"
      />
      <UButton
        data-testid="node-breakpoint"
        class="nodrag nopan"
        :icon="breakpoint ? 'i-tabler-circle-filled' : 'i-tabler-circle'"
        :color="breakpoint ? 'error' : 'neutral'"
        variant="ghost"
        size="xs"
        :aria-label="
          breakpoint ? t('workflow.debug.remove_breakpoint') : t('workflow.debug.add_breakpoint')
        "
        :aria-pressed="breakpoint"
        @pointerdown.stop
        @click.stop="emit('toggle-breakpoint')"
      />
      <span
        v-if="visualState.showRunStatus"
        data-testid="node-run-status"
        class="flex items-center gap-1 text-[9px] font-medium"
        :class="runStatusText"
      >
        <span class="size-1.5 rounded-full" :class="runStatusDot" aria-hidden="true" />
        {{ t(`workflow.node.run_${runStatus}`) }}
      </span>
      <span
        v-if="debugCurrent"
        data-testid="node-debug-current"
        class="flex items-center gap-1 text-[9px] font-medium text-warning"
      >
        <span class="size-1.5 animate-pulse rounded-full bg-warning motion-reduce:animate-none" />
        {{ t('workflow.debug.current') }}
      </span>
    </header>

    <div class="grid grid-cols-2 gap-x-6 px-3 py-2 text-[11px]">
      <div class="space-y-1.5">
        <div v-for="pin in leftPins" :key="pin.key" class="relative flex h-5 items-center">
          <Handle
            :id="graphHandle(pin.channel, 'input', pin.id)"
            type="target"
            :position="Position.Left"
            :class="pin.channel === 'data' ? 'workflow-handle-data' : 'workflow-handle-signal'"
            :style="{ backgroundColor: pin.color }"
          />
          <span class="truncate text-toned">{{ pin.id }}</span>
        </div>
      </div>
      <div class="space-y-1.5 text-right">
        <div
          v-for="pin in rightPins"
          :key="pin.key"
          class="relative flex h-5 items-center justify-end"
        >
          <span class="truncate" :class="pin.channel === 'error' ? 'text-error' : 'text-toned'">
            {{ pin.id }}
          </span>
          <Handle
            :id="graphHandle(pin.channel, 'output', pin.id)"
            type="source"
            :position="Position.Right"
            :class="pin.channel === 'data' ? 'workflow-handle-data' : 'workflow-handle-signal'"
            :style="{ backgroundColor: pin.color }"
          />
        </div>
      </div>
    </div>

    <div
      v-if="surface.inlineInputs.length"
      class="nodrag nopan space-y-2 border-t border-default bg-muted/15 px-3 py-2.5"
      @pointerdown.stop
    >
      <div v-for="item in surface.inlineInputs" :key="item.key" class="space-y-1">
        <div class="flex items-center gap-2 text-[10px]">
          <span class="font-medium text-toned">{{ portTitle(item.port) }}</span>
          <span v-if="item.port.unit" class="ml-auto text-dimmed">{{ item.port.unit }}</span>
        </div>
        <WorkflowValueEditor
          :adapter="item.editorAdapter"
          :port="item.port"
          :model-value="inputValue(item.port)"
          :target-slot="targetSlot"
          compact
          @update:model-value="
            emit('command', {
              kind: 'bind-value',
              nodeId: node.id,
              portId: item.port.id,
              value: $event,
            })
          "
        />
      </div>
    </div>

    <footer
      v-if="projection.statusEvents.length"
      class="flex items-center gap-1.5 border-t border-default px-3 py-1.5 text-[10px] text-dimmed"
    >
      <UIcon name="i-tabler-eye" class="size-3" aria-hidden="true" />
      <span class="truncate">{{
        projection.statusEvents.map((event) => event.code).join(', ')
      }}</span>
    </footer>
  </article>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import { useI18n } from 'vue-i18n'
import type { EditorCommand, Node, NodeProjection } from '@/app/editor/EditorSession'
import type { PortProjection } from '../../../../contracts/node/3.1/authoring-projection'
import { graphHandle, type HandleChannel } from '@/app/editor/graphHandles'
import type { NodeRunStatus } from '@/app/editor/runTrace'
import type { DiagnosticSeverity } from '@/app/editor/workflowDiagnostics'
import { workflowNodeVisualState } from '@/app/editor/workflowNodeVisualState'
import { projectAuthoringSurface } from '@/app/editor/authoringSurface'

const WorkflowValueEditor = defineAsyncComponent(
  () => import('@/app/editor/WorkflowValueEditor.vue'),
)

interface Props {
  node: Node
  projection: NodeProjection
  selected?: boolean
  runStatus?: NodeRunStatus
  breakpoint?: boolean
  debugCurrent?: boolean
  diagnosticSeverity?: DiagnosticSeverity
  connectedInputIds?: ReadonlySet<string>
  targetSlot?: string
}

interface PinView {
  key: string
  id: string
  channel: HandleChannel
  color: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'toggle-breakpoint': []
  command: [command: EditorCommand]
}>()
const { t, te } = useI18n()

const title = computed(() => {
  if (props.node.label) return props.node.label
  if (props.projection.titleKey && te(props.projection.titleKey))
    return t(props.projection.titleKey)
  return props.node.nodeRef.nodeTypeId.split('/').filter(Boolean).at(-2) ?? props.node.id
})

const iconName = computed(() => `i-tabler-${props.projection.icon || 'box'}`)

const visualState = computed(() => workflowNodeVisualState(props))
const surface = computed(() =>
  projectAuthoringSurface(props.projection, props.node, props.connectedInputIds ?? new Set()),
)
const diagnosticIcon = computed(() =>
  visualState.value.diagnosticTone === 'error'
    ? 'i-tabler-alert-circle-filled'
    : visualState.value.diagnosticTone === 'warning'
      ? 'i-tabler-alert-triangle-filled'
      : 'i-tabler-info-circle-filled',
)
const diagnosticClass = computed(() => {
  if (visualState.value.diagnosticTone === 'error') return 'text-error'
  if (visualState.value.diagnosticTone === 'warning') return 'text-warning'
  return 'text-info'
})
const runStatusText = computed(() => {
  if (props.runStatus === 'failed') return 'text-error'
  if (props.runStatus === 'cancelled' || props.runStatus === 'routed') return 'text-warning'
  return 'text-primary'
})
const runStatusDot = computed(() => [
  props.runStatus === 'failed' && 'bg-error',
  (props.runStatus === 'cancelled' || props.runStatus === 'routed') && 'bg-warning',
  props.runStatus === 'running' && 'bg-primary animate-pulse motion-reduce:animate-none',
])

const leftPins = computed<PinView[]>(() => [
  ...props.projection.signals
    .filter((signal) => signal.direction === 'input')
    .map((signal) => ({
      key: `signal:${signal.channel}:${signal.id}`,
      id: signal.id,
      channel: signal.channel,
      color: signal.channel === 'error' ? '#f87171' : '#a1a1aa',
    })),
  ...props.projection.dataInputs.map((port) => ({
    key: `data:${port.id}`,
    id: port.id,
    channel: 'data' as const,
    color: port.type.color || '#a1a1aa',
  })),
])

const rightPins = computed<PinView[]>(() => [
  ...props.projection.signals
    .filter((signal) => signal.direction === 'output')
    .map((signal) => ({
      key: `signal:${signal.channel}:${signal.id}`,
      id: signal.id,
      channel: signal.channel,
      color: signal.channel === 'error' ? '#f87171' : '#a1a1aa',
    })),
  ...props.projection.dataOutputs.map((port) => ({
    key: `data:${port.id}`,
    id: port.id,
    channel: 'data' as const,
    color: port.type.color || '#a1a1aa',
  })),
])

function inputValue(port: PortProjection): unknown {
  const binding = props.node.bindings[port.id]
  return binding?.kind === 'value' ? binding.value : port.default
}

function portTitle(port: PortProjection): string {
  return port.titleKey && te(port.titleKey) ? t(port.titleKey) : port.id
}
</script>

<style scoped>
.workflow-node :deep(.vue-flow__handle) {
  width: 9px;
  height: 9px;
  border: 2px solid var(--ui-bg-elevated);
}

.workflow-node :deep(.vue-flow__handle-left) {
  left: -0.75rem;
}

.workflow-node :deep(.vue-flow__handle-right) {
  right: -0.75rem;
}

.workflow-node :deep(.workflow-handle-signal) {
  width: 10px;
  height: 10px;
  border-radius: 2px;
}

.workflow-node :deep(.workflow-handle-signal.vue-flow__handle-left) {
  transform: translate(-50%, -50%) rotate(45deg);
}

.workflow-node :deep(.workflow-handle-signal.vue-flow__handle-right) {
  transform: translate(50%, -50%) rotate(45deg);
}
</style>
