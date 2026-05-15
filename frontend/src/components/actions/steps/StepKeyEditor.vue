<template>
  <div class="flex items-center gap-2 text-xs">
    <span class="text-dimmed">按键</span>
    <input
      type="text"
      placeholder="W / Space / Ctrl"
      class="input-flat w-28"
      :value="step.vk ?? ''"
      @change="onUpdate('vk', ($event.target as HTMLInputElement).value.trim())"
    />
    <template v-if="step.kind !== 'key_up'">
      <span class="text-dimmed ml-1">时长</span>
      <input
        type="number"
        min="0"
        step="10"
        class="input-flat w-20"
        :value="step.durationMs ?? 50"
        @change="onUpdate('durationMs', Number(($event.target as HTMLInputElement).value))"
      />
      <span class="text-dimmed">ms</span>
    </template>
  </div>
</template>

<script setup lang="ts">
import type { Step } from '@/stores/actions'

defineProps<{ step: Step }>()
const emit = defineEmits<{ update: [Partial<Step>] }>()

function onUpdate(field: keyof Step, v: string | number) {
  emit('update', { [field]: v } as Partial<Step>)
}
</script>
