<template>
  <aside class="flex h-full w-full min-w-0 flex-col border-l border-default bg-default">
    <div class="flex items-center gap-2 border-b border-default px-4 py-3">
      <div class="min-w-0 flex-1">
        <h2 class="truncate text-sm font-semibold text-highlighted">
          {{ t('workflow.graphs.call_inspector') }}
        </h2>
        <p class="truncate font-mono text-[10px] text-dimmed">{{ call.id }} → {{ graph.id }}</p>
      </div>
      <UButton
        icon="i-tabler-arrow-up-right"
        color="neutral"
        variant="ghost"
        size="xs"
        :aria-label="t('workflow.graphs.open')"
        @click="emit('open')"
      />
      <UButton
        icon="i-tabler-trash"
        color="error"
        variant="ghost"
        size="xs"
        :aria-label="t('common.delete')"
        @click="emit('remove')"
      />
    </div>

    <div class="flex-1 space-y-6 overflow-y-auto p-4">
      <section class="space-y-2">
        <label :for="`graph-call-label-${call.id}`" class="block text-xs font-medium text-toned">
          {{ t('workflow.inspector.label') }}
        </label>
        <UInput
          :id="`graph-call-label-${call.id}`"
          :model-value="call.label || ''"
          :placeholder="graph.name || graph.id"
          class="w-full"
          @change="setLabel"
        />
        <p class="text-[11px] leading-5 text-muted">{{ t('workflow.graphs.call_hint') }}</p>
      </section>

      <section v-if="ports.length" class="space-y-3">
        <h3 class="text-xs font-semibold text-highlighted">
          {{ t('workflow.inspector.inputs') }}
        </h3>
        <WorkflowInputBindingEditor
          v-for="port in ports"
          :key="port.id"
          :node="editorNode"
          :port="port"
          @command="applyBindingCommand"
        />
      </section>
      <div v-else class="rounded-lg border border-default bg-elevated/30 px-3 py-4">
        <p class="text-xs text-muted">{{ t('workflow.graphs.no_call_inputs') }}</p>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Graph, GraphCall, Node } from '../../../../contracts/workflow/current/workflow-source'
import type { PortProjection } from '../../../../contracts/node/current/authoring-projection'
import type { EditorCommand } from './EditorSession'
import WorkflowInputBindingEditor from './WorkflowInputBindingEditor.vue'

const props = defineProps<{
  call: GraphCall
  graph: Graph
  ports: PortProjection[]
}>()
const emit = defineEmits<{ update: [call: GraphCall]; open: []; remove: [] }>()
const { t } = useI18n()

const editorNode = computed<Node>(() => ({
  id: props.call.id,
  nodeRef: {
    nodeTypeId: 'https://schemas.yotta.dev/nodes/graph-call',
    version: '1.0.0',
    semanticDigest: `sha256:${'0'.repeat(64)}`,
  },
  position: props.call.position,
  config: {},
  bindings: props.call.bindings,
}))

function setLabel(event: Event): void {
  emit('update', { ...props.call, label: (event.target as HTMLInputElement).value })
}

function applyBindingCommand(command: EditorCommand): void {
  const bindings = { ...props.call.bindings }
  switch (command.kind) {
    case 'bind-value':
      bindings[command.portId] = { kind: 'value', value: command.value }
      break
    case 'bind-blob':
      bindings[command.portId] = { kind: 'blob', blob: command.blob }
      break
    case 'bind-default':
      bindings[command.portId] = { kind: 'default' }
      break
    case 'clear-binding':
      delete bindings[command.portId]
      break
    default:
      return
  }
  emit('update', { ...props.call, bindings })
}
</script>
