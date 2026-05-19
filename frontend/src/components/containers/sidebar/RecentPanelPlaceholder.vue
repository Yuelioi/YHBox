<!-- frontend/src/components/containers/sidebar/RecentPanelPlaceholder.vue -->
<template>
  <SidebarSection
    title="最近"
    icon="i-tabler-clock"
    title-color="indigo"
    :count="0"
    :expanded="expanded"
    @update:expanded="$emit('update:expanded', $event)"
  >
    <p class="text-[10px] text-dimmed italic px-1">
      最近使用节点 (LRU, max 8).
    </p>
  </SidebarSection>
</template>

<script setup lang="ts">
import SidebarSection from './SidebarSection.vue'

defineProps<{ expanded: boolean }>()
defineEmits<{ 'update:expanded': [v: boolean] }>()
</script>
