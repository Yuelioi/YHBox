<template>
  <!-- Inline pin literal — type-polymorphic input.
       NodeInspector 用它编辑未连接 data-in pin 的 config.literal[pinName]. -->
  <UInput
    v-if="type === 'number'"
    type="number"
    :model-value="modelValue ?? 0"
    size="xs"
    @update:model-value="(v: any) => emit('update:modelValue', Number(v) || 0)"
  />
  <UCheckbox
    v-else-if="type === 'bool'"
    :model-value="!!modelValue"
    @update:model-value="(v: boolean) => emit('update:modelValue', v)"
  />
  <UInput
    v-else
    :model-value="modelValue == null ? '' : String(modelValue)"
    size="xs"
    :placeholder="placeholder"
    @update:model-value="(v: any) => emit('update:modelValue', String(v))"
  />
</template>

<script setup lang="ts">
import type { PinType } from '../pinSpec'

defineProps<{
  type: PinType
  modelValue: any
  placeholder?: string
}>()
const emit = defineEmits<{ (e: 'update:modelValue', v: any): void }>()
</script>
