<template>
  <div class="flex items-center gap-2 text-xs flex-wrap">
    <select
      class="input-flat w-16"
      :value="step.mouseButton ?? 'left'"
      @change="onUpdate('mouseButton', ($event.target as HTMLSelectElement).value)"
    >
      <option value="left">左键</option>
      <option value="middle">中键</option>
      <option value="right">右键</option>
    </select>
    <span class="text-dimmed">从</span>
    <input
      type="number"
      min="0"
      max="1"
      step="0.001"
      class="input-flat w-16"
      :value="step.xRatio ?? 0.3"
      @change="onUpdate('xRatio', toNum($event))"
    />
    <input
      type="number"
      min="0"
      max="1"
      step="0.001"
      class="input-flat w-16"
      :value="step.yRatio ?? 0.5"
      @change="onUpdate('yRatio', toNum($event))"
    />
    <span class="text-dimmed">到</span>
    <input
      type="number"
      min="0"
      max="1"
      step="0.001"
      class="input-flat w-16"
      :value="step.x2Ratio ?? 0.7"
      @change="onUpdate('x2Ratio', toNum($event))"
    />
    <input
      type="number"
      min="0"
      max="1"
      step="0.001"
      class="input-flat w-16"
      :value="step.y2Ratio ?? 0.5"
      @change="onUpdate('y2Ratio', toNum($event))"
    />
    <span class="text-dimmed">时长</span>
    <input
      type="number"
      min="0"
      step="50"
      class="input-flat w-20"
      :value="step.durationMs ?? 300"
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
function onUpdate(field: keyof Step, v: string | number) {
  emit('update', { [field]: v } as Partial<Step>)
}
</script>
