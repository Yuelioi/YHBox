<script setup lang="ts">
import { computed } from 'vue'
import type { InputSpec } from '@bindings/yhbox/internal/node'
const props = defineProps<{ input: InputSpec; value: any }>()
const emit = defineEmits<{ (e: 'update', v: { x: number; y: number }): void }>()
const pt = computed(() => props.value ?? { x: 0, y: 0 })
function update(field: 'x' | 'y', v: number) {
  emit('update', { ...pt.value, [field]: v })
}
</script>
<template>
  <div class="flex items-center gap-2">
    <span class="text-xs text-toned">X</span>
    <UInputNumber size="sm" :model-value="pt.x" :step="0.01"
      @update:model-value="(v: number) => update('x', v)" />
    <span class="text-xs text-toned">Y</span>
    <UInputNumber size="sm" :model-value="pt.y" :step="0.01"
      @update:model-value="(v: number) => update('y', v)" />
  </div>
</template>
