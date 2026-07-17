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
            color="neutral"
            :disabled="!canAddVariable"
            :aria-label="t('workflow.inspector.state_add')"
            @click="addStateVariable"
          />
        </div>
      </section>

      <div v-if="variables.length" class="space-y-2">
        <div
          v-for="variable in variables"
          :key="variable.name"
          class="flex items-center gap-2 rounded-lg border border-default bg-elevated/35 px-3 py-2.5"
        >
          <span class="min-w-0 flex-1 truncate font-mono text-xs text-toned">{{
            variable.name
          }}</span>
          <span class="max-w-28 truncate text-[10px] text-dimmed">{{
            variableTypeLabel(variable)
          }}</span>
          <UButton
            icon="i-tabler-trash"
            color="error"
            variant="ghost"
            size="xs"
            :aria-label="t('workflow.inspector.state_remove', { name: variable.name })"
            @click="emit('command', { kind: 'remove-state-variable', name: variable.name })"
          />
        </div>
      </div>

      <div v-else class="rounded-lg border border-dashed border-default px-4 py-8 text-center">
        <UIcon name="i-tabler-database" class="mx-auto mb-2 size-6 text-dimmed" />
        <p class="text-xs text-muted">{{ t('workflow.state_panel.empty') }}</p>
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

const props = defineProps<{ variables: Variable[]; types: TypeProjection[] }>()
const emit = defineEmits<{ command: [command: EditorCommand]; close: [] }>()
const { t, te } = useI18n()
const newVariableName = ref('')
const newVariableTypeId = ref('')
const stateTypes = computed(() =>
  props.types.filter((type) =>
    type.representations.some((representation) => representation.kind === 'inline-json'),
  ),
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

function variableTypeLabel(variable: Variable): string {
  if (variable.type.kind !== 'ref') return variable.type.kind
  const typeId = variable.type.ref.typeId
  const type = props.types.find((candidate) => candidate.typeRef.typeId === typeId)
  if (type?.titleKey && te(type.titleKey)) return t(type.titleKey)
  return typeId.split('/').at(-2) ?? typeId
}
</script>
