<template>
  <div class="flex items-center gap-2 text-xs">
    <span class="text-dimmed">休眠</span>
    <input
      type="number"
      min="0"
      step="50"
      class="input-flat w-24"
      :value="step.durationMs ?? 100"
      @change="emit('update', { durationMs: Number(($event.target as HTMLInputElement).value) })"
    />
    <span class="text-dimmed">ms</span>
  </div>
</template>

<script setup lang="ts">
import type { Step } from '@/stores/actions'

defineProps<{ step: Step }>()
const emit = defineEmits<{ update: [Partial<Step>] }>()
</script>
