<template>
  <div class="flex items-center gap-2 text-xs">
    <span class="text-dimmed">滚动</span>
    <input
      type="number"
      step="1"
      class="input-flat w-20"
      :value="step.scrollDelta ?? 3"
      @change="emit('update', { scrollDelta: Number(($event.target as HTMLInputElement).value) })"
    />
    <span class="text-dimmed">
      格 ({{ (step.scrollDelta ?? 0) > 0 ? '上推' : (step.scrollDelta ?? 0) < 0 ? '下推' : '0' }})
    </span>
  </div>
</template>

<script setup lang="ts">
import type { Step } from '@/stores/actions'

defineProps<{ step: Step }>()
const emit = defineEmits<{ update: [Partial<Step>] }>()
</script>
