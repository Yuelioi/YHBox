<template>
  <UFormField
    :label="label"
    :description="description"
    :required="field.required"
    :ui="inspectorFieldUI"
    class="w-full"
  >
    <USelectMenu
      v-if="selectItems !== undefined"
      :model-value="modelValue"
      :items="selectItems"
      :placeholder="selectPlaceholder"
      :search-input="{ placeholder: t('workflow.inspector.search_target') }"
      :virtualize="selectItems.length > 40"
      value-key="value"
      label-key="label"
      class="w-full"
      @update:model-value="emit('update:modelValue', $event)"
    />
    <p
      v-if="selectItems !== undefined && selectItems.length === 0"
      class="mt-1 text-xs text-warning"
    >
      {{ t('workflow.inspector.no_installed_target') }}
    </p>
    <template v-if="selectItems === undefined">
      <USelectMenu
        v-if="field.control === 'state-variable'"
        :model-value="modelValue"
        :items="stateVariableItems"
        :search-input="{ placeholder: t('workflow.state_panel.search') }"
        :virtualize="stateVariableItems.length > 40"
        value-key="value"
        label-key="label"
        class="w-full"
        @update:model-value="emit('update:modelValue', $event)"
      />
      <USelect
        v-else-if="field.control === 'select'"
        :model-value="modelValue"
        :items="enumItems"
        value-key="value"
        label-key="label"
        class="w-full"
        @update:model-value="emit('update:modelValue', $event)"
      />
      <USwitch
        v-else-if="field.control === 'toggle'"
        :model-value="Boolean(modelValue)"
        @update:model-value="emit('update:modelValue', $event)"
      />
      <UInput
        v-else-if="field.control === 'number' || field.control === 'integer'"
        :model-value="numberValue"
        type="number"
        :step="field.control === 'integer' ? 1 : 'any'"
        :min="numericConstraint(field.constraints.minimum)"
        :max="numericConstraint(field.constraints.maximum)"
        class="w-full"
        @update:model-value="updateNumber"
      />
      <StructuredOutputFieldsEditor
        v-else-if="field.editorAdapter === 'structured-output-fields'"
        :model-value="modelValue"
        @update:model-value="emit('update:modelValue', $event)"
      />
      <UTextarea
        v-else-if="field.control === 'code'"
        :model-value="typeof modelValue === 'string' ? modelValue : ''"
        :rows="12"
        :maxlength="field.constraints.maxLength"
        spellcheck="false"
        class="w-full font-mono text-xs leading-relaxed"
        @update:model-value="emit('update:modelValue', $event)"
      />
      <UTextarea
        v-else-if="jsonControl"
        v-model="jsonText"
        :rows="5"
        class="w-full font-mono text-xs"
        @blur="commitJson"
      />
      <UInput
        v-else
        :model-value="typeof modelValue === 'string' ? modelValue : ''"
        :placeholder="stringDefault"
        :maxlength="field.constraints.maxLength"
        class="w-full"
        @update:model-value="emit('update:modelValue', $event)"
      />
    </template>
    <p v-if="jsonError" class="mt-1 text-xs text-error">{{ jsonError }}</p>
  </UFormField>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { FieldProjection } from '@/contracts/node'
import StructuredOutputFieldsEditor from './StructuredOutputFieldsEditor.vue'

const props = defineProps<{
  field: FieldProjection
  modelValue: unknown
  stateVariables?: string[]
  selectItems?: Array<{ label: string; value: string }>
  selectPlaceholder?: string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: unknown] }>()
const { t, te } = useI18n()

const label = computed(() => {
  if (props.field.titleKey && te(props.field.titleKey)) return t(props.field.titleKey)
  return props.field.title || props.field.id
})
const description = computed(() => {
  if (props.field.descriptionKey && te(props.field.descriptionKey))
    return t(props.field.descriptionKey)
  return props.field.description || ''
})
const inspectorFieldUI = {
  labelWrapper: 'items-start',
  label: 'min-w-0 text-xs font-medium text-toned',
  description: 'mt-1 text-[11px] leading-4 text-muted',
  container: 'mt-2',
}
const enumItems = computed(() =>
  props.field.constraints.enum.map((value) => ({ label: String(value), value })),
)
const stateVariableItems = computed(() =>
  (props.stateVariables ?? []).map((name) => ({ label: name, value: name })),
)
const jsonControl = computed(() => ['json', 'object', 'list'].includes(props.field.control))
const numberValue = computed(() =>
  typeof props.modelValue === 'number' ? props.modelValue : undefined,
)
const stringDefault = computed(() =>
  props.field.hasDefault && typeof props.field.default === 'string'
    ? props.field.default
    : undefined,
)
const jsonText = ref(formatJson(props.modelValue))
const jsonError = ref('')

watch(
  () => props.modelValue,
  (value) => {
    jsonText.value = formatJson(value)
    jsonError.value = ''
  },
)

function updateNumber(value: string | number): void {
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) return
  emit('update:modelValue', props.field.control === 'integer' ? Math.trunc(parsed) : parsed)
}

function commitJson(): void {
  try {
    emit('update:modelValue', JSON.parse(jsonText.value))
    jsonError.value = ''
  } catch (error) {
    jsonError.value = error instanceof Error ? error.message : String(error)
  }
}

function numericConstraint(value: unknown): number | undefined {
  return typeof value === 'number' ? value : undefined
}

function formatJson(value: unknown): string {
  if (value === undefined) return ''
  return JSON.stringify(value, null, 2)
}
</script>
