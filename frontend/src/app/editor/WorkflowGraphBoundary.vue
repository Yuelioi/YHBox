<template>
  <article
    data-testid="workflow-graph-boundary"
    class="min-w-[180px] overflow-visible rounded-lg border border-dashed border-primary/50 bg-default/95 shadow-sm"
  >
    <header class="flex items-center gap-2 border-b border-dashed border-default px-3 py-2">
      <UIcon :name="icon" class="size-4 text-primary" />
      <div class="min-w-0 flex-1">
        <p class="truncate text-xs font-semibold text-highlighted">{{ title }}</p>
        <p class="text-[10px] text-dimmed">{{ $t('workflow.graphs.boundary_authoring') }}</p>
      </div>
    </header>

    <div v-if="boundary.role === 'entry'" class="space-y-1.5 px-3 py-2 text-[11px]">
      <div class="relative flex h-5 items-center justify-end gap-2">
        <span class="text-toned">in</span>
        <Handle
          :id="graphHandle('exec', 'output', 'in')"
          type="source"
          :position="Position.Right"
          class="workflow-handle-signal"
        />
      </div>
      <div
        v-for="port in boundary.inputs"
        :key="port.id"
        class="relative flex h-5 items-center justify-end"
      >
        <span class="max-w-36 truncate text-toned">{{ port.id }}</span>
        <Handle
          :id="graphHandle('data', 'output', port.id)"
          type="source"
          :position="Position.Right"
          class="workflow-handle-data"
        />
      </div>
    </div>

    <div v-else-if="boundary.role === 'exit'" class="px-3 py-2 text-[11px]">
      <div class="relative flex h-5 items-center gap-2">
        <Handle
          :id="graphHandle(boundary.exit!.channel, 'input', 'in')"
          type="target"
          :position="Position.Left"
          class="workflow-handle-signal"
        />
        <span :class="boundary.exit!.channel === 'error' ? 'text-error' : 'text-toned'">
          {{ boundary.exit!.id }}
        </span>
      </div>
    </div>

    <div v-else class="space-y-1.5 px-3 py-2 text-[11px]">
      <div
        v-for="port in boundary.outputs"
        :key="port.id"
        class="relative flex h-5 items-center gap-2"
      >
        <Handle
          :id="graphHandle('data', 'input', port.id)"
          type="target"
          :position="Position.Left"
          class="workflow-handle-data"
        />
        <span class="max-w-36 truncate text-toned">{{ port.id }}</span>
      </div>
    </div>
  </article>
</template>

<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { graphHandle } from './graphHandles'
import type { GraphBoundaryNodeData } from './workflowGraphBoundary'

const props = defineProps<{ boundary: GraphBoundaryNodeData }>()
const { t } = useI18n()

const title = computed(() => {
  if (props.boundary.role === 'entry') return t('workflow.graphs.boundary_entry')
  if (props.boundary.role === 'output') return t('workflow.graphs.boundary_output')
  return t('workflow.graphs.boundary_exit')
})
const icon = computed(() => {
  if (props.boundary.role === 'entry') return 'i-tabler-login-2'
  if (props.boundary.role === 'output') return 'i-tabler-brackets-contain-end'
  return props.boundary.exit?.channel === 'error' ? 'i-tabler-alert-triangle' : 'i-tabler-logout-2'
})
</script>
