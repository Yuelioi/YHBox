<template>
  <section
    data-testid="workflow-graph-manager"
    class="flex h-full min-h-0 flex-col bg-default"
    :aria-label="t('workflow.graphs.manager')"
  >
    <header class="flex items-center gap-3 border-b border-default px-3 py-3">
      <div class="min-w-0 flex-1">
        <h2 class="text-sm font-semibold text-highlighted">
          {{ t('workflow.graphs.manager') }}
        </h2>
        <p class="mt-0.5 text-[10px] text-muted">
          {{ t('workflow.graphs.manager_hint') }}
        </p>
      </div>
      <UButton
        data-testid="workflow-graph-new"
        icon="i-tabler-plus"
        size="xs"
        color="neutral"
        variant="ghost"
        :label="t('workflow.graphs.new')"
        @click="createDefinition"
      />
    </header>

    <div class="border-b border-default p-3">
      <UInput
        v-model="query"
        icon="i-tabler-search"
        size="sm"
        class="w-full"
        :placeholder="t('workflow.graphs.search')"
        :aria-label="t('workflow.graphs.search')"
      />
    </div>

    <div class="min-h-0 flex-1 overflow-y-auto p-2">
      <div v-if="definitions.length" class="space-y-1">
        <article
          v-for="definition in definitions"
          :key="definition.id"
          class="rounded-lg border transition-colors"
          :class="
            definition.id === currentGraphId
              ? 'border-primary/50 bg-primary/5'
              : 'border-transparent hover:border-default hover:bg-elevated/40'
          "
        >
          <div class="flex min-w-0 items-center gap-1 p-1">
            <button
              type="button"
              :data-testid="`workflow-graph-definition-${definition.id}`"
              :draggable="canCallDefinition(definition)"
              class="min-w-0 flex-1 rounded-md px-2 py-1.5 text-left focus-visible:outline-2 focus-visible:outline-primary"
              :class="canCallDefinition(definition) ? 'cursor-grab active:cursor-grabbing' : ''"
              :title="
                canCallDefinition(definition)
                  ? t('workflow.graphs.drag_call_hint')
                  : definition.kind === 'subgraph'
                    ? t('workflow.graphs.call_unavailable')
                    : undefined
              "
              @dragstart="startDefinitionDrag($event, definition)"
              @click="openDefinition(definition.id)"
            >
              <span class="flex min-w-0 items-center gap-2">
                <UIcon
                  :name="
                    definition.kind === 'main'
                      ? 'i-tabler-home'
                      : canCallDefinition(definition)
                        ? 'i-tabler-grip-vertical'
                        : 'i-tabler-folders'
                  "
                  class="size-4 shrink-0"
                  :class="definition.id === currentGraphId ? 'text-primary' : 'text-dimmed'"
                />
                <span class="min-w-0 flex-1 truncate text-xs font-medium text-highlighted">
                  {{ definition.kind === 'main' ? t('workflow.graphs.main') : definition.name }}
                </span>
                <span
                  v-if="definition.kind === 'subgraph'"
                  class="shrink-0 rounded bg-elevated px-1.5 py-0.5 text-[9px] text-muted"
                >
                  {{ t('workflow.graphs.call_count', { count: definition.callCount }) }}
                </span>
              </span>
              <span class="mt-1 flex min-w-0 items-center gap-2 pl-6 text-[9px] text-muted">
                <code
                  v-if="definition.duplicateName || definition.name === definition.id"
                  class="max-w-32 truncate"
                  >{{ definition.shortId }}</code
                >
                <span v-if="definition.kind === 'subgraph'" class="truncate">
                  {{
                    t('workflow.graphs.interface_summary', {
                      inputs: definition.dataInputCount,
                      outputs: definition.dataOutputCount,
                      exits: definition.execExitCount + definition.errorExitCount,
                    })
                  }}
                </span>
                <UIcon
                  v-if="!definition.interfaceHealthy"
                  name="i-tabler-alert-triangle"
                  class="size-3 shrink-0 text-warning"
                  :title="t('workflow.graphs.interface_unhealthy')"
                />
              </span>
            </button>

            <div v-if="definition.kind === 'subgraph'" class="flex shrink-0 items-center gap-0.5">
              <UButton
                data-testid="workflow-graph-insert-call"
                icon="i-tabler-library-plus"
                color="primary"
                variant="ghost"
                size="xs"
                :label="t('workflow.graphs.call')"
                :disabled="!canCallDefinition(definition)"
                :aria-label="t('workflow.graphs.call_named', { name: definition.name })"
                :title="
                  canCallDefinition(definition)
                    ? t('workflow.graphs.call_named', { name: definition.name })
                    : t('workflow.graphs.call_unavailable')
                "
                @click="insertCall(definition.id)"
              />
              <UDropdownMenu :items="definitionActions(definition)">
                <UButton
                  icon="i-tabler-dots-vertical"
                  color="neutral"
                  variant="ghost"
                  size="xs"
                  :aria-label="t('workflow.graphs.definition_actions', { name: definition.name })"
                />
              </UDropdownMenu>
            </div>
          </div>
        </article>
      </div>
      <div v-else class="px-4 py-8 text-center text-xs text-muted">
        {{ t('workflow.graphs.search_empty') }}
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GraphDefinitionSource, GraphDefinitionSummary } from './subgraphManagement'
import { projectGraphDefinitions } from './subgraphManagement'

const props = withDefaults(
  defineProps<{
    source: GraphDefinitionSource
    currentGraphId?: string
    callableGraphIds?: string[]
    dragFormat?: string
  }>(),
  {
    callableGraphIds: () => [],
    dragFormat: 'application/x-yotta-graph-call',
  },
)
const emit = defineEmits<{
  open: [graphId: string]
  insert: [graphId: string]
  create: []
  rename: [graphId: string]
  duplicate: [graphId: string]
  delete: [graphId: string]
  deleteCascade: [graphId: string]
  locate: [parentGraphId: string, callId: string]
}>()
const { t } = useI18n()
const query = ref('')
const definitions = computed(() => projectGraphDefinitions(props.source, query.value))

function openDefinition(graphId: string): void {
  emit('open', graphId)
}

function createDefinition(): void {
  emit('create')
}

function canCallDefinition(definition: GraphDefinitionSummary): boolean {
  return definition.kind === 'subgraph' && props.callableGraphIds.includes(definition.id)
}

function startDefinitionDrag(event: DragEvent, definition: GraphDefinitionSummary): void {
  if (!canCallDefinition(definition) || !event.dataTransfer) {
    event.preventDefault()
    return
  }
  event.dataTransfer.effectAllowed = 'copy'
  event.dataTransfer.setData(props.dragFormat, definition.id)
}

function insertCall(graphId: string): void {
  emit('insert', graphId)
}

function definitionActions(definition: GraphDefinitionSummary) {
  const locations = definition.references.length
    ? [
        {
          label: t('workflow.graphs.call_locations'),
          icon: 'i-tabler-map-pin',
          children: definition.references.map((reference) => ({
            label: `${reference.parentGraphName} · ${reference.callLabel}`,
            icon: 'i-tabler-focus-2',
            onSelect: () => emit('locate', reference.parentGraphId, reference.callId),
          })),
        },
      ]
    : []
  return [
    locations,
    [
      {
        label: t('common.rename'),
        icon: 'i-tabler-pencil',
        onSelect: () => emit('rename', definition.id),
      },
      {
        label: t('common.copy'),
        icon: 'i-tabler-copy',
        onSelect: () => emit('duplicate', definition.id),
      },
    ],
    [
      definition.callCount
        ? {
            label: t('common.delete'),
            icon: 'i-tabler-trash-x',
            color: 'error' as const,
            onSelect: () => emit('deleteCascade', definition.id),
          }
        : {
            label: t('common.delete'),
            icon: 'i-tabler-trash',
            color: 'error' as const,
            onSelect: () => emit('delete', definition.id),
          },
    ],
  ].filter((group) => group.length)
}
</script>
