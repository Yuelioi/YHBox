<template>
  <KeyChordValueEditor
    v-if="editorAdapter === 'key-chord'"
    :model-value="keyChordValue"
    @update:model-value="emit('update:model-value', $event)"
  />
  <USwitch
    v-else-if="type?.control === 'toggle'"
    :model-value="Boolean(modelValue)"
    size="sm"
    @update:model-value="emit('update:model-value', $event)"
  />
  <UInputNumber
    v-else-if="type?.control === 'number' || type?.control === 'integer'"
    :model-value="numericValue"
    :min="numericConstraint(type.constraints.minimum)"
    :max="numericConstraint(type.constraints.maximum)"
    :step="type.control === 'integer' ? 1 : 'any'"
    size="sm"
    class="w-full"
    @update:model-value="emit('update:model-value', Number($event))"
  />
  <AdaptiveSelect
    v-else-if="type?.control === 'select'"
    :model-value="selectValue"
    :items="selectItems"
    width-mode="fill"
    size="sm"
    @update:model-value="emit('update:model-value', $event)"
  />
  <UInput
    v-else-if="type?.control === 'text'"
    :model-value="textValue"
    size="sm"
    class="w-full"
    @change="setText"
  />
  <UTextarea
    v-else
    :model-value="jsonValue"
    :rows="2"
    size="sm"
    class="w-full font-mono text-xs"
    @change="setJSON"
  />
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent } from 'vue'
import type { TypeProjection } from '../../../../contracts/node/3.1/authoring-projection'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'

const KeyChordValueEditor = defineAsyncComponent(() => import('./KeyChordValueEditor.vue'))

const props = defineProps<{
  modelValue: unknown
  type?: TypeProjection
  editorAdapter?: 'key-chord'
}>()
const emit = defineEmits<{ 'update:model-value': [value: unknown] }>()

const numericValue = computed(() =>
  typeof props.modelValue === 'number' ? props.modelValue : undefined,
)
const keyChordValue = computed(() =>
  Array.isArray(props.modelValue)
    ? props.modelValue.filter((value): value is string => typeof value === 'string')
    : [],
)
const textValue = computed(() => (typeof props.modelValue === 'string' ? props.modelValue : ''))
const selectValue = computed<string | number | boolean | null>(() => {
  const value = props.modelValue
  return typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean'
    ? value
    : null
})
const selectItems = computed(() =>
  (props.type?.constraints.enum ?? [])
    .filter((value): value is string | number | boolean =>
      ['string', 'number', 'boolean'].includes(typeof value),
    )
    .map((value) => ({ label: String(value), value })),
)
const jsonValue = computed(() => JSON.stringify(props.modelValue ?? null, null, 2))

function numericConstraint(value: unknown): number | undefined {
  return typeof value === 'number' ? value : undefined
}

function setText(event: Event): void {
  emit('update:model-value', (event.target as HTMLInputElement).value)
}

function setJSON(event: Event): void {
  try {
    emit('update:model-value', JSON.parse((event.target as HTMLTextAreaElement).value))
  } catch {
    return
  }
}
</script>
