<template>
  <aside
    data-testid="workflow-graph-interface"
    class="flex h-full w-full min-w-0 flex-col border-l border-default bg-default"
  >
    <div class="flex items-center gap-2 border-b border-default px-4 py-3">
      <UIcon name="i-tabler-plug-connected" class="size-4 text-primary" />
      <div class="min-w-0 flex-1">
        <h2 class="truncate text-sm font-semibold text-highlighted">
          {{ t('workflow.graphs.interface_title') }}
        </h2>
        <p class="truncate font-mono text-[10px] text-dimmed">{{ graph.id }}</p>
      </div>
      <UButton
        icon="i-tabler-refresh"
        color="neutral"
        variant="ghost"
        size="xs"
        :aria-label="t('workflow.graphs.infer_interface')"
        @click="emit('infer')"
      />
    </div>

    <div class="flex-1 space-y-5 overflow-y-auto p-4">
      <p class="text-[11px] leading-5 text-muted">{{ t('workflow.graphs.interface_hint') }}</p>

      <InterfaceSection
        icon="i-tabler-login-2"
        :title="t('workflow.graphs.boundary_entry')"
        :items="entryItems"
      />
      <InterfaceSection
        v-if="graph.inputs.length"
        icon="i-tabler-arrow-bar-to-right"
        :title="t('workflow.inspector.inputs')"
        :items="inputItems"
      />
      <InterfaceSection
        v-if="graph.outputs.length"
        icon="i-tabler-arrow-bar-left"
        :title="t('workflow.graphs.boundary_output')"
        :items="outputItems"
      />
      <InterfaceSection
        v-if="graph.exits?.length"
        icon="i-tabler-logout-2"
        :title="t('workflow.graphs.boundary_exit')"
        :items="exitItems"
      />

      <div
        v-if="!graph.inputs.length && !graph.outputs.length && !graph.exits?.length"
        class="rounded-lg border border-dashed border-default px-3 py-5 text-center"
      >
        <p class="text-xs text-muted">{{ t('workflow.graphs.interface_empty') }}</p>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, resolveComponent, type PropType } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Graph } from '../../../../contracts/workflow/current/workflow-source'

interface InterfaceItem {
  label: string
  endpoint: string
  detail?: string
  tone?: 'error'
}

const props = defineProps<{ graph: Graph }>()
const emit = defineEmits<{ infer: [] }>()
const { t } = useI18n()

const endpoint = (nodeId: string, portId: string) => `${nodeId}.${portId}`
const entryItems = computed<InterfaceItem[]>(() =>
  props.graph.entries?.length
    ? props.graph.entries.map((entry) => ({
        label: 'in',
        endpoint: endpoint(entry.nodeId, entry.portId),
      }))
    : [{ label: 'in', endpoint: '—' }],
)
const inputItems = computed<InterfaceItem[]>(() =>
  props.graph.inputs.map((port) => ({
    label: port.id,
    endpoint: endpoint(port.nodeId, port.portId),
    detail: typeLabel(port.type),
  })),
)
const outputItems = computed<InterfaceItem[]>(() =>
  props.graph.outputs.map((port) => ({
    label: port.id,
    endpoint: endpoint(port.nodeId, port.portId),
    detail: typeLabel(port.type),
  })),
)
const exitItems = computed<InterfaceItem[]>(() =>
  (props.graph.exits ?? []).map((exit) => ({
    label: exit.id,
    endpoint: endpoint(exit.endpoint.nodeId, exit.endpoint.portId),
    detail: exit.channel,
    tone: exit.channel === 'error' ? 'error' : undefined,
  })),
)

const InterfaceSection = defineComponent({
  props: {
    icon: { type: String, required: true },
    title: { type: String, required: true },
    items: { type: Array as PropType<InterfaceItem[]>, required: true },
  },
  setup(section) {
    const UIcon = resolveComponent('UIcon')
    return () =>
      h('section', { class: 'space-y-2' }, [
        h('div', { class: 'flex items-center gap-2' }, [
          h(UIcon, { name: section.icon, class: 'size-3.5 text-dimmed' }),
          h('h3', { class: 'text-xs font-semibold text-highlighted' }, section.title),
        ]),
        h(
          'div',
          { class: 'divide-y divide-default overflow-hidden rounded-lg border border-default' },
          section.items.map((item) =>
            h('div', { class: 'grid grid-cols-[minmax(0,1fr)_auto] gap-2 px-3 py-2.5' }, [
              h('div', { class: 'min-w-0' }, [
                h(
                  'p',
                  {
                    class: `truncate text-xs ${item.tone === 'error' ? 'text-error' : 'text-toned'}`,
                  },
                  item.label,
                ),
                item.detail
                  ? h('p', { class: 'truncate font-mono text-[9px] text-dimmed' }, item.detail)
                  : null,
              ]),
              h('code', { class: 'max-w-40 truncate text-[10px] text-muted' }, item.endpoint),
            ]),
          ),
        ),
      ])
  },
})

function typeLabel(type: Graph['inputs'][number]['type']): string {
  if (type.kind === 'ref') return type.ref.typeId.split('/').at(-2) ?? type.ref.typeId
  if (type.kind === 'variable') return `$${type.variable}`
  if (type.kind === 'list') return `List<${typeLabel(type.element)}>`
  return type.members.map(typeLabel).join(' | ')
}
</script>
