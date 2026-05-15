<template>
  <div class="flex items-center gap-2 text-xs">
    <span class="text-dimmed">移到 x</span>
    <input
      type="number"
      min="0"
      max="1"
      step="0.001"
      class="input-flat w-20"
      :value="step.xRatio ?? 0.5"
      @change="onUpdate('xRatio', toNum($event))"
    />
    <span class="text-dimmed ml-1">y</span>
    <input
      type="number"
      min="0"
      max="1"
      step="0.001"
      class="input-flat w-20"
      :value="step.yRatio ?? 0.5"
      @change="onUpdate('yRatio', toNum($event))"
    />
  </div>
</template>

<script setup lang="ts">
import type { Step } from '@/stores/actions'

defineProps<{ step: Step }>()
const emit = defineEmits<{ update: [Partial<Step>] }>()

function toNum(e: Event): number {
  return Number((e.target as HTMLInputElement).value)
}
function onUpdate(field: 'xRatio' | 'yRatio', v: number) {
  emit('update', { [field]: v })
}
</script>
