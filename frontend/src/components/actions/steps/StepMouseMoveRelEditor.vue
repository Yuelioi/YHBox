<template>
  <div class="flex items-center gap-2 text-xs flex-wrap">
    <span class="text-dimmed">视角 dx</span>
    <input
      type="number"
      step="10"
      class="input-flat w-20"
      :value="step.dx ?? 100"
      @change="onUpdate('dx', toNum($event))"
    />
    <span class="text-dimmed">dy</span>
    <input
      type="number"
      step="10"
      class="input-flat w-20"
      :value="step.dy ?? 0"
      @change="onUpdate('dy', toNum($event))"
    />
    <span class="text-dimmed">时长</span>
    <input
      type="number"
      min="0"
      step="50"
      class="input-flat w-20"
      :value="step.durationMs ?? 200"
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
