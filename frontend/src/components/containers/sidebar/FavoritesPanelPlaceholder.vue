<!-- frontend/src/components/containers/sidebar/FavoritesPanelPlaceholder.vue -->
<!-- Empty shell for Editor v2 B (node discovery spec). A ships layout only. -->
<template>
  <SidebarSection
    title="收藏"
    icon="i-tabler-star"
    title-color="amber"
    :count="0"
    :expanded="expanded"
    @update:expanded="$emit('update:expanded', $event)"
  >
    <p class="text-[10px] text-dimmed italic px-1">
      节点收藏 — Editor v2 B 实装. 拖 ★ 节点 → 画布建实例.
    </p>
  </SidebarSection>
</template>

<script setup lang="ts">
import SidebarSection from './SidebarSection.vue'

defineProps<{ expanded: boolean }>()
defineEmits<{ 'update:expanded': [v: boolean] }>()
</script>
