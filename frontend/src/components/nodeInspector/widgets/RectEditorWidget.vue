<script setup lang="ts">
import { computed } from 'vue'
import type { InputSpec } from '@bindings/yhbox/internal/node'
const props = defineProps<{ input: InputSpec; value: any }>()
const emit = defineEmits<{ (e: 'update', v: { x: number; y: number; w: number; h: number }): void }>()
const r = computed(() => props.value ?? { x: 0, y: 0, w: 0, h: 0 })
function update(field: 'x' | 'y' | 'w' | 'h', v: number) {
  emit('update', { ...r.value, [field]: v })
}
</script>
<template>
  <div class="grid grid-cols-2 gap-2">
    <div class="flex items-center gap-2">
      <span class="text-xs text-toned w-4">X</span>
      <UInputNumber size="sm" :model-value="r.x" :step="0.01"
        @update:model-value="(v: number) => update('x', v)" />
    </div>
    <div class="flex items-center gap-2">
      <span class="text-xs text-toned w-4">Y</span>
      <UInputNumber size="sm" :model-value="r.y" :step="0.01"
        @update:model-value="(v: number) => update('y', v)" />
    </div>
    <div class="flex items-center gap-2">
      <span class="text-xs text-toned w-4">W</span>
      <UInputNumber size="sm" :model-value="r.w" :step="0.01"
        @update:model-value="(v: number) => update('w', v)" />
    </div>
    <div class="flex items-center gap-2">
      <span class="text-xs text-toned w-4">H</span>
      <UInputNumber size="sm" :model-value="r.h" :step="0.01"
        @update:model-value="(v: number) => update('h', v)" />
    </div>
  </div>
</template>
