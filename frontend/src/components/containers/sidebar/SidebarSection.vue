<!-- frontend/src/components/containers/sidebar/SidebarSection.vue -->
<template>
  <div class="border-b border-default">
    <button
      class="w-full flex items-center justify-between gap-2 px-3 py-2 text-xs hover:bg-elevated/50 transition-colors"
      :class="titleColorClass"
      @click="$emit('update:expanded', !expanded)"
    >
      <span class="flex items-center gap-2">
        <UIcon v-if="icon" :name="icon" class="size-3.5" />
        <span class="font-medium">{{ title }}</span>
        <span v-if="count !== undefined" class="text-dimmed text-[10px]">({{ count }})</span>
      </span>
      <span class="flex items-center gap-1">
        <slot name="header-action" />
        <UIcon
          :name="expanded ? 'i-tabler-chevron-down' : 'i-tabler-chevron-right'"
          class="size-3.5 text-dimmed"
        />
      </span>
    </button>
    <div v-show="expanded" class="p-2 space-y-1">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  title: string
  expanded: boolean
  icon?: string
  count?: number
  titleColor?: 'amber' | 'indigo' | 'emerald' | 'default'
}>()

defineEmits<{
  'update:expanded': [v: boolean]
}>()

const titleColorClass = computed(() => {
  switch (props.titleColor) {
    case 'amber': return 'text-amber-400'
    case 'indigo': return 'text-indigo-400'
    case 'emerald': return 'text-emerald-400'
    default: return 'text-default'
  }
})
</script>
