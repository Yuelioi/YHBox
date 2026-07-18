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
          v-for="variable in visibleVariables"
          :key="variable.name"
          class="rounded-lg border border-default bg-elevated/35"
        >
          <div
            draggable="true"
            class="flex cursor-grab items-center gap-2 px-3 py-2.5 active:cursor-grabbing"
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
              icon="i-tabler-edit"
              color="neutral"
              variant="ghost"
              size="xs"
              :title="
                referenceCount(variable.name)
                  ? t('workflow.state_panel.type_change_referenced')
                  : undefined
              "
              :aria-label="t('workflow.state_panel.type_change', { name: variable.name })"
              @click="beginTypeChange(variable)"
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
          <div
            v-if="editingName === variable.name"
            class="space-y-2 border-t border-default px-3 py-2"
          >
            <div class="flex items-center gap-2">
              <USelect
                v-model="editingTypeId"
                class="min-w-0 flex-1"
                :items="stateTypeItems"
                value-key="value"
                label-key="label"
                size="sm"
              />
              <UButton
                color="neutral"
                variant="ghost"
                size="xs"
                :label="t('common.cancel')"
                @click="cancelTypeChange"
              />
              <UButton
                size="xs"
                :label="t('common.confirm')"
                :disabled="!editingTypeId || editingImpact.issues.length > 0"
                @click="commitTypeChange(variable.name)"
              />
            </div>
            <div v-if="referenceCount(variable.name)" class="rounded-md bg-warning/10 p-2">
              <p class="text-[11px] leading-4 text-warning">
                {{
                  t('workflow.state_panel.type_change_impact', {
                    count: referenceCount(variable.name),
                  })
                }}
              </p>
              <div class="mt-2 max-h-36 space-y-1 overflow-y-auto">
                <UButton
                  v-for="reference in referencesFor(variable.name)"
                  :key="`${reference.graphId}:${reference.nodeId}`"
                  color="neutral"
                  variant="ghost"
                  size="xs"
                  class="w-full justify-start font-mono"
                  icon="i-tabler-focus-2"
                  :label="`${reference.graphId} / ${reference.nodeId} · ${reference.mode}`"
                  @click="emit('locate-reference', reference.graphId, reference.nodeId)"
                />
              </div>
            </div>
            <div
              v-if="editingImpact.issues.length"
              class="rounded-md border border-error/30 bg-error/10 p-2"
            >
              <p class="text-[11px] leading-4 text-error">
                {{
                  t('workflow.state_panel.type_change_blocked', {
                    count: editingImpact.issues.length,
                  })
                }}
              </p>
              <UButton
                v-for="issue in editingImpact.issues"
                :key="`${issue.graphId}:${issue.edge.from.nodeId}:${issue.edge.from.portId}:${issue.edge.to.nodeId}:${issue.edge.to.portId}`"
                color="neutral"
                variant="ghost"
                size="xs"
                class="mt-1 w-full justify-start font-mono"
                icon="i-tabler-plug-connected-x"
                :label="`${issue.graphId} · ${issue.edge.from.nodeId}.${issue.edge.from.portId} → ${issue.edge.to.nodeId}.${issue.edge.to.portId} · ${issueDispositionLabel(issue.disposition)}`"
                @click="emit('locate-reference', issue.graphId, issue.edge.to.nodeId)"
              />
            </div>
            <p v-else-if="referenceCount(variable.name)" class="text-[11px] leading-4 text-success">
              {{ t('workflow.state_panel.type_change_safe') }}
            </p>
          </div>
        </div>
        <UButton
          v-if="visibleVariables.length < filteredVariables.length"
          color="neutral"
          variant="soft"
          size="sm"
          class="w-full justify-center"
          :label="
            t('workflow.state_panel.show_more', {
              remaining: filteredVariables.length - visibleVariables.length,
            })
          "
          @click="visibleLimit += STATE_PAGE_SIZE"
        />
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
import type {
  EditorCommand,
  StateReferenceLocation,
  StateTypeChangeImpact,
} from '@/app/editor/EditorSession'

const props = defineProps<{
  variables: Variable[]
  types: TypeProjection[]
  references: Record<string, StateReferenceLocation[]>
  typeChangeImpact: (name: string, typeId: string) => StateTypeChangeImpact
}>()
const emit = defineEmits<{
  command: [command: EditorCommand]
  insert: [name: string, mode: 'read' | 'write']
  locate: [name: string]
  'locate-reference': [graphId: string, nodeId: string]
  close: []
}>()
const { t, te } = useI18n()
const newVariableName = ref('')
const newVariableTypeId = ref('')
const searchQuery = ref('')
const editingName = ref('')
const editingTypeId = ref('')
const STATE_PAGE_SIZE = 100
const visibleLimit = ref(STATE_PAGE_SIZE)
const filteredVariables = computed(() => {
  const query = searchQuery.value.trim().toLocaleLowerCase()
  if (!query) return props.variables
  return props.variables.filter((variable) =>
    [variable.name, variableTypeLabel(variable)].some((value) =>
      value.toLocaleLowerCase().includes(query),
    ),
  )
})
const visibleVariables = computed(() => filteredVariables.value.slice(0, visibleLimit.value))
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
const editingImpact = computed<StateTypeChangeImpact>(() =>
  editingName.value && editingTypeId.value
    ? props.typeChangeImpact(editingName.value, editingTypeId.value)
    : { references: [], issues: [] },
)

watch(
  stateTypes,
  (values) => {
    if (!values.some((type) => type.typeRef.typeId === newVariableTypeId.value))
      newVariableTypeId.value = values[0]?.typeRef.typeId ?? ''
  },
  { immediate: true },
)

watch(searchQuery, () => {
  visibleLimit.value = STATE_PAGE_SIZE
})

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
  return props.references[name]?.length ?? 0
}

function referencesFor(name: string): StateReferenceLocation[] {
  return props.references[name] ?? []
}

function issueDispositionLabel(disposition: 'conversion' | 'incompatible'): string {
  return t(`workflow.state_panel.type_change_${disposition}`)
}

function beginTypeChange(variable: Variable): void {
  editingName.value = variable.name
  editingTypeId.value = variable.type.kind === 'ref' ? variable.type.ref.typeId : ''
}

function cancelTypeChange(): void {
  editingName.value = ''
  editingTypeId.value = ''
}

function commitTypeChange(name: string): void {
  if (editingImpact.value.issues.length) return
  const type = stateTypes.value.find(
    (candidate) => candidate.typeRef.typeId === editingTypeId.value,
  )
  if (!type) return
  emit('command', {
    kind: 'update-state-variable',
    name,
    type: { kind: 'ref', ref: { ...type.typeRef } },
    defaultValue: defaultStateValue(type),
  })
  cancelTypeChange()
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
