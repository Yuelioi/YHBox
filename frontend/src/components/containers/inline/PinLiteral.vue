<template>
  <!--
    v4 §7.1 inline pin literal — type-polymorphic input.
    Used by NodeInspector to edit config.literal[pinName] for unconnected data-in pins.
    Future: also render in ContainerFlowNode next to each data-in pin handle
    (UE-style; deferred — current MVP edits via Inspector instead).
  -->
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
