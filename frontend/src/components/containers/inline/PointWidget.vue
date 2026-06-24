<template>
  <div class="space-y-2">
    <div class="grid grid-cols-2 gap-1.5">
      <div class="space-y-0.5">
        <label class="text-[10px] text-dimmed">X %</label>
        <UInputNumber
          :model-value="displayX"
          size="xs"
          class="w-full"
          :min="0"
          :max="100"
          :step="0.1"
          @update:model-value="(v: number) => onChange('x', v)"
        />
      </div>
      <div class="space-y-0.5">
        <label class="text-[10px] text-dimmed">Y %</label>
        <UInputNumber
          :model-value="displayY"
          size="xs"
          class="w-full"
          :min="0"
          :max="100"
          :step="0.1"
          @update:model-value="(v: number) => onChange('y', v)"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { PointValue } from '@/components/containers/nodeRegistry/index'

const props = defineProps<{
  modelValue: PointValue | null
  fieldPath: string
}>()
const emit = defineEmits<{ (e: 'update:modelValue', v: PointValue): void }>()

function round4(n: number): number {
  return Math.round(n * 1e4) / 1e4
}

const safeValue = computed<PointValue>(() => {
  const v = props.modelValue
  if (!v || typeof v.x !== 'number' || typeof v.y !== 'number') return { x: 0, y: 0 }
  return { x: v.x, y: v.y }
})

const displayX = computed(() => round4(safeValue.value.x * 100))
const displayY = computed(() => round4(safeValue.value.y * 100))

function onChange(field: 'x' | 'y', displayVal: number) {
  const next: PointValue = { ...safeValue.value }
  next[field] = round4(displayVal / 100)
  emit('update:modelValue', next)
}
</script>
