<template>
  <PointValueEditor
    v-if="adapter === 'point'"
    :model-value="modelValue"
    :target-slot="targetSlot"
    :compact="compact"
    @update:model-value="emit('update:model-value', $event)"
  />
  <RegionValueEditor
    v-else-if="adapter === 'region'"
    :model-value="modelValue"
    :target-slot="targetSlot"
    :compact="compact"
    @update:model-value="emit('update:model-value', $event)"
  />
  <ColorRangeValueEditor
    v-else-if="adapter === 'color-range'"
    :model-value="modelValue"
    :target-slot="targetSlot"
    :compact="compact"
    @update:model-value="emit('update:model-value', $event)"
  />
  <DurationValueEditor
    v-else-if="adapter === 'duration'"
    :model-value="modelValue"
    :compact="compact"
    @update:model-value="emit('update:model-value', $event)"
  />
  <KeyChordValueEditor
    v-else-if="adapter === 'key-chord'"
    :model-value="keyChordValue"
    @update:model-value="emit('update:model-value', $event)"
  />
  <USwitch
    v-else-if="adapter === 'toggle'"
    :model-value="Boolean(modelValue)"
    :size="compact ? 'xs' : 'sm'"
    @update:model-value="emit('update:model-value', $event)"
  />
  <UInputNumber
    v-else-if="adapter === 'number'"
    :model-value="numberValue"
    :min="numericConstraint(port.type.constraints.minimum)"
    :max="numericConstraint(port.type.constraints.maximum)"
    :step="port.type.control === 'integer' ? 1 : 'any'"
    :size="compact ? 'xs' : 'sm'"
    class="w-full"
    @update:model-value="emit('update:model-value', Number($event))"
  />
  <AdaptiveSelect
    v-else-if="adapter === 'select'"
    :model-value="selectValue"
    :items="port.type.constraints.enum.map((value) => ({ label: String(value), value }))"
    width-mode="fill"
    :size="compact ? 'xs' : 'sm'"
    class="w-full"
    @update:model-value="emit('update:model-value', $event)"
  />
  <UInput
    v-else-if="adapter === 'text'"
    :model-value="textValue"
    :size="compact ? 'xs' : 'sm'"
    class="w-full"
    @change="setText"
  />
  <UTextarea
    v-else
    :model-value="jsonValue"
    :rows="compact ? 2 : 5"
    :size="compact ? 'xs' : 'sm'"
    class="w-full font-mono text-xs"
    @change="setJSON"
  />
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent } from 'vue'
import type { PortProjection } from '../../../../contracts/node/current/authoring-projection'
import type { ValueEditorAdapter } from './authoringSurface'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'

const PointValueEditor = defineAsyncComponent(() => import('./PointValueEditor.vue'))
const RegionValueEditor = defineAsyncComponent(() => import('./RegionValueEditor.vue'))
const ColorRangeValueEditor = defineAsyncComponent(() => import('./ColorRangeValueEditor.vue'))
const DurationValueEditor = defineAsyncComponent(() => import('./DurationValueEditor.vue'))
const KeyChordValueEditor = defineAsyncComponent(() => import('./KeyChordValueEditor.vue'))

const props = defineProps<{
  adapter: ValueEditorAdapter
  port: PortProjection
  modelValue: unknown
  targetSlot?: string
  compact?: boolean
}>()
const emit = defineEmits<{ 'update:model-value': [value: unknown] }>()
const keyChordValue = computed(() =>
  Array.isArray(props.modelValue)
    ? props.modelValue.filter((value): value is string => typeof value === 'string')
    : [],
)
const numberValue = computed(() =>
  typeof props.modelValue === 'number' ? props.modelValue : undefined,
)
const textValue = computed(() => (typeof props.modelValue === 'string' ? props.modelValue : ''))
const selectValue = computed(() =>
  typeof props.modelValue === 'string' ||
  typeof props.modelValue === 'number' ||
  typeof props.modelValue === 'boolean'
    ? props.modelValue
    : null,
)
const jsonValue = computed(() =>
  props.modelValue === undefined
    ? ''
    : JSON.stringify(props.modelValue, null, props.compact ? 0 : 2),
)

function numericConstraint(value: unknown): number | undefined {
  return typeof value === 'number' ? value : undefined
}

function setJSON(event: Event): void {
  try {
    emit('update:model-value', JSON.parse((event.target as HTMLTextAreaElement).value))
  } catch {
    return
  }
}

function setText(event: Event): void {
  emit('update:model-value', (event.target as HTMLInputElement).value)
}
</script>
