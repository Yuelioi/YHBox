<template>
  <aside
    data-testid="workflow-state-panel"
    class="flex h-full w-[340px] shrink-0 flex-col border-l border-default bg-default"
  >
    <div class="flex items-center justify-between border-b border-default px-4 py-3">
      <div class="min-w-0">
        <h2 class="text-sm font-semibold text-highlighted">
          {{ t('workflow.state_panel.title') }}
        </h2>
        <p class="mt-0.5 text-[10px] text-dimmed">
          {{ t('workflow.state_panel.hint') }}
        </p>
      </div>
      <UButton
        icon="i-tabler-x"
        color="neutral"
        variant="ghost"
        size="xs"
        :aria-label="t('common.close')"
        @click="emit('close')"
      />
    </div>

    <div class="flex-1 space-y-4 overflow-y-auto p-4">
      <section class="space-y-3 rounded-lg border border-default p-3">
        <div class="flex items-center justify-between">
          <h3 class="text-xs font-semibold text-highlighted">
            {{ t('workflow.inspector.state_title') }}
          </h3>
          <UBadge color="neutral" variant="soft" size="sm">{{ variables.length }}</UBadge>
        </div>
        <p class="text-[11px] leading-5 text-muted">
          {{ t('workflow.inspector.state_hint') }}
        </p>
        <div class="grid grid-cols-[1fr_1fr_auto] gap-2">
          <UInput
            v-model="newVariableName"
            :placeholder="t('workflow.inspector.state_name_placeholder')"
            size="sm"
          />
          <USelect
            v-model="newVariableTypeId"
            :items="stateTypeItems"
            value-key="value"
            label-key="label"
            size="sm"
          />
          <UButton
            icon="i-tabler-plus"
            size="sm"
            color="primary"
            :disabled="!canAddVariable"
            :aria-label="t('workflow.inspector.state_add')"
            @click="addStateVariable"
          />
        </div>
      </section>

      <UInput
        v-model="searchQuery"
        icon="i-tabler-search"
        size="sm"
        :placeholder="t('workflow.state_panel.search')"
      />

      <div v-if="filteredVariables.length" class="space-y-2">
        <div
          v-for="variable in filteredVariables"
          :key="variable.name"
          draggable="true"
          class="flex cursor-grab items-center gap-2 rounded-lg border border-default bg-elevated/35 px-3 py-2.5 active:cursor-grabbing"
          :title="t('workflow.state_panel.drag_hint')"
          @dragstart="startStateDrag($event, variable.name)"
        >
          <UIcon name="i-tabler-grip-vertical" class="size-4 shrink-0 text-dimmed" />
          <span class="min-w-0 flex-1 truncate font-mono text-xs text-toned">{{
            variable.name
          }}</span>
          <span class="max-w-28 truncate text-[10px] text-dimmed">{{
            variableTypeLabel(variable)
          }}</span>
          <UButton
            v-if="referenceCount(variable.name)"
            icon="i-tabler-focus-2"
            color="neutral"
            variant="soft"
            size="xs"
            :label="String(referenceCount(variable.name))"
            :aria-label="
              t('workflow.state_panel.locate_references', {
                name: variable.name,
                count: referenceCount(variable.name),
              })
            "
            @click="emit('locate', variable.name)"
          />
          <UButton
            icon="i-tabler-database-export"
            color="neutral"
            variant="ghost"
            size="xs"
            :aria-label="t('workflow.state_panel.insert_read', { name: variable.name })"
            @click="emit('insert', variable.name, 'read')"
          />
          <UButton
            icon="i-tabler-database-import"
            color="neutral"
            variant="ghost"
            size="xs"
            :aria-label="t('workflow.state_panel.insert_write', { name: variable.name })"
            @click="emit('insert', variable.name, 'write')"
          />
          <UButton
            icon="i-tabler-trash"
            color="error"
            variant="ghost"
            size="xs"
            :disabled="referenceCount(variable.name) > 0"
            :title="
              referenceCount(variable.name)
                ? t('workflow.state_panel.remove_referenced')
                : undefined
            "
            :aria-label="t('workflow.inspector.state_remove', { name: variable.name })"
            @click="emit('command', { kind: 'remove-state-variable', name: variable.name })"
          />
        </div>
      </div>

      <div v-else class="rounded-lg border border-dashed border-default px-4 py-8 text-center">
        <UIcon name="i-tabler-database" class="mx-auto mb-2 size-6 text-dimmed" />
        <p class="text-xs text-muted">
          {{
            variables.length
              ? t('workflow.state_panel.no_results')
              : t('workflow.state_panel.empty')
          }}
        </p>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Variable } from '../../../../contracts/workflow/3.1/workflow-source'
import type { TypeProjection } from '../../../../contracts/node/3.1/authoring-projection'
import type { EditorCommand } from '@/app/editor/EditorSession'

const props = defineProps<{
  variables: Variable[]
  types: TypeProjection[]
  references: Record<string, number>
}>()
const emit = defineEmits<{
  command: [command: EditorCommand]
  insert: [name: string, mode: 'read' | 'write']
  locate: [name: string]
  close: []
}>()
const { t, te } = useI18n()
const newVariableName = ref('')
const newVariableTypeId = ref('')
const searchQuery = ref('')
const filteredVariables = computed(() => {
  const query = searchQuery.value.trim().toLocaleLowerCase()
  if (!query) return props.variables
  return props.variables.filter((variable) =>
    [variable.name, variableTypeLabel(variable)].some((value) =>
      value.toLocaleLowerCase().includes(query),
    ),
  )
})
const stateTypes = computed(() =>
  props.types.filter((type) => type.traits.includes('durable') && hasValidDefault(type)),
)
const stateTypeItems = computed(() =>
  stateTypes.value.map((type) => ({
    label:
      type.titleKey && te(type.titleKey)
        ? t(type.titleKey)
        : type.typeRef.typeId.split('/').at(-2)!,
    value: type.typeRef.typeId,
  })),
)
const selectedStateType = computed(() =>
  stateTypes.value.find((type) => type.typeRef.typeId === newVariableTypeId.value),
)
const canAddVariable = computed(
  () =>
    /^[A-Za-z0-9_][A-Za-z0-9._-]*$/.test(newVariableName.value) && Boolean(selectedStateType.value),
)

watch(
  stateTypes,
  (values) => {
    if (!values.some((type) => type.typeRef.typeId === newVariableTypeId.value))
      newVariableTypeId.value = values[0]?.typeRef.typeId ?? ''
  },
  { immediate: true },
)

function addStateVariable(): void {
  const type = selectedStateType.value
  if (!type || !canAddVariable.value) return
  emit('command', {
    kind: 'add-state-variable',
    name: newVariableName.value,
    type: { kind: 'ref', ref: { ...type.typeRef } },
    defaultValue: defaultStateValue(type),
  })
  newVariableName.value = ''
}

function defaultStateValue(type: TypeProjection): unknown {
  if (type.examples.length) return structuredClone(type.examples[0])
  switch (type.control) {
    case 'text':
      return ''
    case 'number':
    case 'integer':
      return 0
    case 'toggle':
      return false
    case 'select':
      return type.constraints.enum[0] ?? null
    case 'list':
      return []
    case 'object':
      return {}
    default:
      return null
  }
}

function hasValidDefault(type: TypeProjection): boolean {
  return type.examples.length > 0 || type.control !== 'object'
}

function variableTypeLabel(variable: Variable): string {
  if (variable.type.kind !== 'ref') return variable.type.kind
  const typeId = variable.type.ref.typeId
  const type = props.types.find((candidate) => candidate.typeRef.typeId === typeId)
  if (type?.titleKey && te(type.titleKey)) return t(type.titleKey)
  return typeId.split('/').at(-2) ?? typeId
}

function referenceCount(name: string): number {
  return props.references[name] ?? 0
}

function startStateDrag(event: DragEvent, name: string): void {
  if (!event.dataTransfer) return
  event.dataTransfer.effectAllowed = 'copy'
  event.dataTransfer.setData(
    'application/x-yotta-state-reference',
    JSON.stringify({ name, mode: event.altKey ? 'write' : 'read' }),
  )
}
</script>
