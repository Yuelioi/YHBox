<!-- Point type 行内编辑 — 用 2 个 number input 横排, 比 JSON textarea 易用. -->
<template>
  <div class="flex items-center gap-1">
    <span class="text-[11px] text-dimmed">x:</span>
    <input
      type="number"
      step="0.01"
      :value="point.x"
      aria-label="X"
      class="w-12 bg-elevated/50 border border-default rounded px-1 py-0.5 text-[11px] focus:border-primary focus:outline-none"
      @input="onChange('x', $event)"
    />
    <span class="text-[11px] text-dimmed">y:</span>
    <input
      type="number"
      step="0.01"
      :value="point.y"
      aria-label="Y"
      class="w-12 bg-elevated/50 border border-default rounded px-1 py-0.5 text-[11px] focus:border-primary focus:outline-none"
      @input="onChange('y', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface PointValue {
  x: number
  y: number
}

const props = defineProps<{ modelValue: PointValue | null | undefined }>()
const emit = defineEmits<{ 'update:modelValue': [v: PointValue] }>()

const point = computed<PointValue>(() => {
  const v = props.modelValue ?? { x: 0, y: 0 }
  return { x: v.x ?? 0, y: v.y ?? 0 }
})

function onChange(axis: 'x' | 'y', e: Event) {
  const next = parseFloat((e.target as HTMLInputElement).value)
  if (Number.isNaN(next)) return
  emit('update:modelValue', { ...point.value, [axis]: next })
}
</script>
