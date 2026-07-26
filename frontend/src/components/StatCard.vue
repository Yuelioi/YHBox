<template>
  <div class="rounded-xl bg-default border border-default p-4">
    <!-- Top row: label + optional icon -->
    <div class="flex items-center">
      <span class="text-xs uppercase tracking-wider text-dimmed">{{ label }}</span>
      <UIcon v-if="icon" :name="icon" class="size-4 text-dimmed ml-auto" />
    </div>
    <!-- Bottom row: value -->
    <div
      class="mt-2 text-3xl font-semibold"
      :class="[tabular !== false ? 'tabular-nums' : '', valueColorClass]"
    >
      {{ value }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  label: string
  value: string | number
  color?: 'default' | 'emerald' | 'violet' | 'amber' | 'sky'
  icon?: string
  tabular?: boolean
}>()

const valueColorClass = computed(() => {
  switch (props.color) {
    case 'emerald':
      return 'text-primary'
    case 'violet':
      return 'text-violet-400'
    case 'amber':
      return 'text-warning'
    case 'sky':
      return 'text-sky-400'
    default:
      return 'text-highlighted'
  }
})
</script>
