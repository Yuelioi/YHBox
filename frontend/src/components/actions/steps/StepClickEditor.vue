<template>
  <div class="flex items-center gap-2 text-xs">
    <span class="text-dimmed">x</span>
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
    <span class="text-dimmed ml-1">时长</span>
    <input
      type="number"
      min="0"
      step="10"
      class="input-flat w-20"
      :value="step.durationMs ?? 50"
      @change="onUpdate('durationMs', toNum($event))"
    />
    <span class="text-dimmed">ms</span>
  </div>
</template>

<script setup lang="ts">
import type { Step } from '@/stores/actions'

defineProps<{ step: Step }>()
const emit = defineEmits<{ update: [Partial<Step>] }>()

function toNum(e: Event): number {
  return Number((e.target as HTMLInputElement).value)
}
function onUpdate(field: keyof Step, v: number) {
  emit('update', { [field]: v } as Partial<Step>)
}
</script>
