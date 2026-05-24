<script setup lang="ts">
import { computed } from 'vue'
import type { InputSpec } from '@bindings/yhbox/internal/node'
const props = defineProps<{ input: InputSpec; value: any }>()
const emit = defineEmits<{ (e: 'update', v: number): void }>()
const min = computed(() => Number(props.input.widget?.props?.min ?? 0))
const max = computed(() => Number(props.input.widget?.props?.max ?? 100))
const step = computed(() => Number(props.input.widget?.props?.step ?? 1))
</script>
<template>
  <div class="flex items-center gap-3">
    <USlider
      class="flex-1"
      :model-value="Number(value ?? min)"
      :min="min"
      :max="max"
      :step="step"
      @update:model-value="(v: number) => emit('update', v)"
    />
    <span class="text-xs text-toned w-12 text-right">{{ value ?? min }}</span>
  </div>
</template>
