<template>
  <article
    data-testid="workflow-graph-call"
    class="min-w-[230px] overflow-visible rounded-lg border bg-default shadow-sm"
    :class="
      selected
        ? 'border-default ring-2 ring-primary/80 ring-offset-2 ring-offset-default'
        : 'border-default'
    "
    @dblclick.stop="emit('open')"
  >
    <header
      class="workflow-node-drag-handle flex cursor-grab items-center gap-2 rounded-t-lg border-b border-default bg-primary/5 px-3 py-2.5"
    >
      <UIcon name="i-tabler-folders" class="size-4 text-primary" />
      <div class="min-w-0 flex-1">
        <p class="truncate text-xs font-semibold text-highlighted">
          {{ call.label || graph.name || graph.id }}
        </p>
        <p class="truncate font-mono text-[10px] text-dimmed">{{ graph.id }}</p>
      </div>
      <UIcon name="i-tabler-arrow-up-right" class="size-3.5 text-dimmed" />
    </header>
    <div class="grid grid-cols-2 gap-x-6 px-3 py-2 text-[11px]">
      <div class="space-y-1.5">
        <div class="relative flex h-5 min-w-0 items-center pl-3">
          <Handle
            :id="graphHandle('exec', 'input', 'in')"
            type="target"
            :position="Position.Left"
            class="workflow-handle-signal"
          />
          <span class="min-w-0 flex-1 truncate text-toned">in</span>
        </div>
        <div
          v-for="port in graph.inputs"
          :key="port.id"
          class="relative flex h-5 min-w-0 items-center pl-3"
        >
          <Handle
            :id="graphHandle('data', 'input', port.id)"
            type="target"
            :position="Position.Left"
            class="workflow-handle-data"
          />
          <span class="min-w-0 flex-1 truncate text-toned">{{
            graphInterfacePortLabel(port)
          }}</span>
        </div>
      </div>
      <div class="space-y-1.5 text-right">
        <div
          v-for="port in graph.outputs"
          :key="port.id"
          class="relative flex h-5 min-w-0 items-center justify-end pr-3"
        >
          <span class="min-w-0 flex-1 truncate text-toned">{{
            graphInterfacePortLabel(port)
          }}</span>
          <Handle
            :id="graphHandle('data', 'output', port.id)"
            type="source"
            :position="Position.Right"
            class="workflow-handle-data"
          />
        </div>
        <div
          v-for="exit in graph.exits ?? []"
          :key="exit.id"
          class="relative flex h-5 min-w-0 items-center justify-end pr-3"
        >
          <span
            class="min-w-0 flex-1 truncate"
            :class="exit.channel === 'error' ? 'text-error' : 'text-toned'"
          >
            {{ graphInterfaceExitLabel(exit) }}
          </span>
          <Handle
            :id="graphHandle(exit.channel, 'output', exit.id)"
            type="source"
            :position="Position.Right"
            class="workflow-handle-signal"
          />
        </div>
      </div>
    </div>
  </article>
</template>

<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import type { Graph, GraphCall } from '../../../../contracts/workflow/current/workflow-source'
import { graphHandle } from './graphHandles'
import { graphInterfaceExitLabel, graphInterfacePortLabel } from './subgraphInterface'

defineProps<{ call: GraphCall; graph: Graph; selected?: boolean }>()
const emit = defineEmits<{ open: [] }>()
</script>
