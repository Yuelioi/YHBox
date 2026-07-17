<template>
  <article
    class="workflow-node min-w-[230px] overflow-visible rounded-lg border bg-elevated shadow-sm transition-[border-color,box-shadow] duration-150"
    :class="selected ? 'border-primary/70 shadow-primary/10' : 'border-default'"
  >
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
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import { useI18n } from 'vue-i18n'
import type { Node, NodeProjection } from '@/app/editor/EditorSession'
import { graphHandle, type HandleChannel } from '@/app/editor/graphHandles'

interface Props {
  node: Node
  projection: NodeProjection
  selected?: boolean
}

interface PinView {
  key: string
  id: string
  channel: HandleChannel
  color: string
}

const props = defineProps<Props>()
const { t, te } = useI18n()

const title = computed(() => {
  if (props.node.label) return props.node.label
  if (props.projection.titleKey && te(props.projection.titleKey))
    return t(props.projection.titleKey)
  return props.node.nodeRef.nodeTypeId.split('/').filter(Boolean).at(-2) ?? props.node.id
})

const iconName = computed(() => `i-tabler-${props.projection.icon || 'box'}`)

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
