<!-- 非折叠分区头: icon + 句式标题 + 可选计数 + actions 槽。
     可折叠场景用既有 SidebarSection, 不归此组件。 -->
<template>
  <div class="flex items-center gap-2 border-b border-default px-3 py-2">
    <UIcon v-if="icon" :name="icon" class="size-4 text-muted" />
    <span class="text-[13px] font-medium text-toned">{{ title }}</span>
    <span v-if="count !== undefined" class="text-xs tabular-nums text-dimmed">({{ count }})</span>
    <div class="flex-1" />
    <slot name="actions" />
  </div>
</template>

<script setup lang="ts">
defineProps<{ title: string; icon?: string; count?: number }>()
</script>
