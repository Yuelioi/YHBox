<template>
  <UFormField :label="label" :hint="hint" class="w-full">
    <USelect
      v-if="field.control === 'state-variable'"
      :model-value="modelValue"
      :items="stateVariableItems"
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
    <p v-if="jsonError" class="mt-1 text-xs text-error">{{ jsonError }}</p>
  </UFormField>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { FieldProjection } from '@/contracts/node'

const props = defineProps<{
  field: FieldProjection
  modelValue: unknown
  stateVariables?: string[]
}>()
const emit = defineEmits<{ 'update:modelValue': [value: unknown] }>()
const { t, te } = useI18n()

const label = computed(() => {
  if (props.field.titleKey && te(props.field.titleKey)) return t(props.field.titleKey)
  return props.field.title || props.field.id
})
const hint = computed(() => {
  if (props.field.descriptionKey && te(props.field.descriptionKey))
    return t(props.field.descriptionKey)
  return props.field.description || t(props.field.required ? 'common.required' : 'common.optional')
})
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
