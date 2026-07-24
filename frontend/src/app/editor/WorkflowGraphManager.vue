<template>
  <UPopover v-model:open="open" mode="click" :ui="{ content: 'w-[390px] p-0' }">
    <UButton
      data-testid="workflow-graph-manager-trigger"
      icon="i-tabler-folders"
      color="neutral"
      variant="ghost"
      size="xs"
      :label="t('workflow.graphs.manager')"
    />

    <template #content>
      <section data-testid="workflow-graph-manager" class="flex max-h-[560px] flex-col">
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
            icon="i-tabler-plus"
            size="xs"
            color="primary"
            variant="soft"
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
              <div class="flex min-w-0 items-start gap-1 p-1">
                <button
                  type="button"
                  class="min-w-0 flex-1 rounded-md px-2 py-1.5 text-left focus-visible:outline-2 focus-visible:outline-primary"
                  @click="openDefinition(definition.id)"
                >
                  <span class="flex min-w-0 items-center gap-2">
                    <UIcon
                      :name="definition.kind === 'main' ? 'i-tabler-home' : 'i-tabler-folders'"
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

                <div v-if="definition.kind === 'subgraph'" class="flex shrink-0 items-center">
                  <UButton
                    icon="i-tabler-pencil"
                    color="neutral"
                    variant="ghost"
                    size="xs"
                    :aria-label="t('workflow.graphs.rename_definition')"
                    @click="renameDefinition(definition.id)"
                  />
                  <UButton
                    icon="i-tabler-trash"
                    color="error"
                    variant="ghost"
                    size="xs"
                    :disabled="definition.callCount > 0"
                    :aria-label="t('workflow.graphs.delete_definition')"
                    :title="
                      definition.callCount
                        ? t('workflow.graphs.delete_definition_referenced', {
                            count: definition.callCount,
                          })
                        : t('workflow.graphs.delete_definition')
                    "
                    @click="deleteDefinition(definition.id)"
                  />
                </div>
              </div>

              <div v-if="definition.references.length" class="border-t border-default/70 px-3 py-2">
                <p class="mb-1.5 text-[9px] font-medium tracking-wide text-dimmed uppercase">
                  {{ t('workflow.graphs.call_locations') }}
                </p>
                <div class="flex flex-wrap gap-1">
                  <button
                    v-for="reference in definition.references"
                    :key="`${reference.parentGraphId}:${reference.callId}`"
                    type="button"
                    class="max-w-full truncate rounded-md bg-elevated px-2 py-1 text-[10px] text-toned hover:bg-accented focus-visible:outline-2 focus-visible:outline-primary"
                    @click="locateCall(reference.parentGraphId, reference.callId)"
                  >
                    {{ reference.parentGraphName }} · {{ reference.callLabel }}
                  </button>
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
  </UPopover>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GraphDefinitionSource } from './subgraphManagement'
import { projectGraphDefinitions } from './subgraphManagement'

const props = defineProps<{ source: GraphDefinitionSource; currentGraphId?: string }>()
const emit = defineEmits<{
  open: [graphId: string]
  create: []
  rename: [graphId: string]
  delete: [graphId: string]
  locate: [parentGraphId: string, callId: string]
}>()
const { t } = useI18n()
const open = ref(false)
const query = ref('')
const definitions = computed(() => projectGraphDefinitions(props.source, query.value))

function openDefinition(graphId: string): void {
  open.value = false
  emit('open', graphId)
}

function createDefinition(): void {
  open.value = false
  emit('create')
}

function renameDefinition(graphId: string): void {
  open.value = false
  emit('rename', graphId)
}

function deleteDefinition(graphId: string): void {
  open.value = false
  emit('delete', graphId)
}

function locateCall(parentGraphId: string, callId: string): void {
  open.value = false
  emit('locate', parentGraphId, callId)
}
</script>
