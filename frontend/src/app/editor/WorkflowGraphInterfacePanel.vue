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
        icon="i-tabler-wand"
        color="neutral"
        variant="ghost"
        size="xs"
        :aria-label="t('workflow.graphs.infer_interface')"
        :title="t('workflow.graphs.infer_interface_hint')"
        @click="emit('infer')"
      />
    </div>

    <div class="flex-1 space-y-5 overflow-y-auto p-4">
      <p class="text-[11px] leading-5 text-muted">{{ t('workflow.graphs.interface_hint') }}</p>

      <section class="space-y-2">
        <div class="flex items-center gap-2">
          <UIcon name="i-tabler-login-2" class="size-3.5 text-dimmed" />
          <h3 class="min-w-0 flex-1 text-xs font-semibold text-highlighted">
            {{ t('workflow.graphs.boundary_entry') }}
          </h3>
          <UDropdownMenu :items="candidateMenuItems('entry')">
            <UButton
              icon="i-tabler-plug-connected"
              color="neutral"
              variant="ghost"
              size="xs"
              :disabled="!availableCandidates('entry').length"
              :aria-label="t('workflow.graphs.bind_entry')"
              :title="
                availableCandidates('entry').length
                  ? t('workflow.graphs.bind_entry')
                  : t('workflow.graphs.no_available_endpoint')
              "
            />
          </UDropdownMenu>
        </div>
        <div class="overflow-hidden rounded-lg border border-default">
          <div class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2 px-3 py-2.5">
            <div class="min-w-0">
              <p class="truncate text-xs text-toned">in</p>
              <code class="block truncate text-[9px] text-dimmed">{{ entryEndpoint }}</code>
            </div>
            <UButton
              v-if="graph.entries?.length"
              icon="i-tabler-unlink"
              color="error"
              variant="ghost"
              size="xs"
              :aria-label="t('workflow.graphs.unbind_entry')"
              @click="emit('remove', 'entry', '')"
            />
          </div>
        </div>
      </section>

      <section v-for="section in sections" :key="section.kind" class="space-y-2">
        <div class="flex items-center gap-2">
          <UIcon :name="section.icon" class="size-3.5 text-dimmed" />
          <h3 class="min-w-0 flex-1 text-xs font-semibold text-highlighted">
            {{ section.title }}
          </h3>
          <UDropdownMenu :items="candidateMenuItems(section.kind)">
            <UButton
              icon="i-tabler-plus"
              color="neutral"
              variant="ghost"
              size="xs"
              :disabled="!availableCandidates(section.kind).length"
              :aria-label="t('workflow.graphs.add_interface_item', { item: section.title })"
              :title="
                availableCandidates(section.kind).length
                  ? t('workflow.graphs.add_interface_item', { item: section.title })
                  : t('workflow.graphs.no_available_endpoint')
              "
            />
          </UDropdownMenu>
        </div>

        <div
          v-if="section.items.length"
          class="divide-y divide-default overflow-hidden rounded-lg border border-default"
        >
          <div v-for="(item, index) in section.items" :key="item.id" class="space-y-2 px-3 py-2.5">
            <div class="flex min-w-0 items-center gap-1">
              <input
                class="min-w-0 flex-1 rounded border border-transparent bg-transparent px-1 py-0.5 text-xs text-toned outline-none hover:border-default focus:border-primary"
                :value="item.name"
                :aria-label="t('workflow.graphs.interface_name')"
                @change="renameItem(section.kind, item.id, $event)"
                @keydown.enter="($event.target as HTMLInputElement).blur()"
              />
              <span
                v-if="item.referenceCount"
                class="shrink-0 rounded bg-warning/10 px-1.5 py-0.5 text-[9px] text-warning"
                :title="
                  t('workflow.graphs.interface_referenced', {
                    count: item.referenceCount,
                  })
                "
              >
                {{ item.referenceCount }}
              </span>
              <UButton
                icon="i-tabler-chevron-up"
                color="neutral"
                variant="ghost"
                size="xs"
                :disabled="index === 0"
                :aria-label="t('workflow.graphs.move_interface_up')"
                @click="emit('move', section.kind, item.id, -1)"
              />
              <UButton
                icon="i-tabler-chevron-down"
                color="neutral"
                variant="ghost"
                size="xs"
                :disabled="index === section.items.length - 1"
                :aria-label="t('workflow.graphs.move_interface_down')"
                @click="emit('move', section.kind, item.id, 1)"
              />
              <UButton
                icon="i-tabler-trash"
                color="error"
                variant="ghost"
                size="xs"
                :disabled="item.referenceCount > 0"
                :aria-label="t('workflow.graphs.remove_interface_item')"
                :title="
                  item.referenceCount
                    ? t('workflow.graphs.interface_referenced', {
                        count: item.referenceCount,
                      })
                    : t('workflow.graphs.remove_interface_item')
                "
                @click="emit('remove', section.kind, item.id)"
              />
            </div>
            <div class="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] gap-2">
              <p
                v-if="item.detail"
                class="truncate font-mono text-[9px]"
                :class="item.tone === 'error' ? 'text-error' : 'text-dimmed'"
              >
                {{ item.detail }}
              </p>
              <code class="max-w-40 truncate text-[9px] text-muted">{{ item.endpoint }}</code>
            </div>
            <p
              v-if="item.name !== item.id"
              class="truncate font-mono text-[8px] text-dimmed"
              :title="item.id"
            >
              ID {{ item.id }}
            </p>
          </div>
        </div>

        <div
          v-else
          class="rounded-lg border border-dashed border-default px-3 py-4 text-center text-[10px] text-muted"
        >
          {{ t('workflow.graphs.interface_section_empty') }}
        </div>
      </section>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Graph } from '../../../../contracts/workflow/current/workflow-source'
import {
  graphInterfaceExitLabel,
  graphInterfacePortLabel,
  type GraphInterfaceCandidate,
  type GraphInterfaceCandidateKind,
  type GraphInterfaceItemKind,
} from './subgraphInterface'

interface InterfaceItem {
  id: string
  name: string
  endpoint: string
  detail?: string
  tone?: 'error'
  referenceCount: number
}

interface InterfaceSection {
  kind: GraphInterfaceItemKind
  icon: string
  title: string
  items: InterfaceItem[]
}

const props = defineProps<{
  graph: Graph
  candidates: GraphInterfaceCandidate[]
  referenceCounts?: Record<string, number>
}>()
const emit = defineEmits<{
  infer: []
  add: [candidateKey: string]
  rename: [kind: GraphInterfaceItemKind, id: string, name: string]
  move: [kind: GraphInterfaceItemKind, id: string, direction: -1 | 1]
  remove: [kind: GraphInterfaceCandidateKind, id: string]
}>()
const { t } = useI18n()

const endpoint = (nodeId: string, portId: string) => `${nodeId}.${portId}`
const entryEndpoint = computed(() => {
  const entry = props.graph.entries?.[0]
  return entry ? endpoint(entry.nodeId, entry.portId) : t('workflow.graphs.interface_unbound')
})
const sections = computed<InterfaceSection[]>(() => [
  {
    kind: 'input',
    icon: 'i-tabler-arrow-bar-to-right',
    title: t('workflow.inspector.inputs'),
    items: props.graph.inputs.map((port) => ({
      id: port.id,
      name: graphInterfacePortLabel(port),
      endpoint: endpoint(port.nodeId, port.portId),
      detail: typeLabel(port.type),
      referenceCount: referenceCount('input', port.id),
    })),
  },
  {
    kind: 'output',
    icon: 'i-tabler-arrow-bar-left',
    title: t('workflow.graphs.boundary_output'),
    items: props.graph.outputs.map((port) => ({
      id: port.id,
      name: graphInterfacePortLabel(port),
      endpoint: endpoint(port.nodeId, port.portId),
      detail: typeLabel(port.type),
      referenceCount: referenceCount('output', port.id),
    })),
  },
  {
    kind: 'exit',
    icon: 'i-tabler-logout-2',
    title: t('workflow.graphs.boundary_exit'),
    items: (props.graph.exits ?? []).map((exit) => ({
      id: exit.id,
      name: graphInterfaceExitLabel(exit),
      endpoint: endpoint(exit.endpoint.nodeId, exit.endpoint.portId),
      detail: exit.channel,
      tone: exit.channel === 'error' ? 'error' : undefined,
      referenceCount: referenceCount('exit', exit.id),
    })),
  },
])

function availableCandidates(kind: GraphInterfaceCandidateKind): GraphInterfaceCandidate[] {
  return props.candidates.filter((candidate) => candidate.kind === kind && !candidate.published)
}

function candidateMenuItems(kind: GraphInterfaceCandidateKind) {
  return [
    availableCandidates(kind).map((candidate) => ({
      label: `${candidate.elementLabel} · ${candidate.name}`,
      icon: candidate.channel === 'error' ? 'i-tabler-alert-triangle' : 'i-tabler-plug',
      onSelect: () => emit('add', candidate.key),
    })),
  ]
}

function referenceCount(kind: GraphInterfaceItemKind, id: string): number {
  return props.referenceCounts?.[`${kind}:${id}`] ?? 0
}

function renameItem(kind: GraphInterfaceItemKind, id: string, event: Event): void {
  emit('rename', kind, id, (event.target as HTMLInputElement).value)
}

function typeLabel(type: Graph['inputs'][number]['type']): string {
  if (type.kind === 'ref') return type.ref.typeId.split('/').at(-2) ?? type.ref.typeId
  if (type.kind === 'variable') return `$${type.variable}`
  if (type.kind === 'list') return `List<${typeLabel(type.element)}>`
  return type.members.map(typeLabel).join(' | ')
}
</script>
