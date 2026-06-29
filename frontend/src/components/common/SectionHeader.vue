<!-- 非折叠分区头: icon + 标题 (xs/600 大写 tracking) + 可选计数 + actions 槽。
     可折叠场景用既有 SidebarSection, 不归此组件。 -->
<template>
  <div class="flex items-center gap-2 border-b border-default px-3 py-2">
    <UIcon v-if="icon" :name="icon" class="size-3.5 text-dimmed" />
    <span class="text-xs font-semibold uppercase tracking-wider text-dimmed">{{ title }}</span>
    <span v-if="count !== undefined" class="text-[10px] text-dimmed">({{ count }})</span>
    <div class="flex-1" />
    <slot name="actions" />
  </div>
</template>

<script setup lang="ts">
defineProps<{ title: string; icon?: string; count?: number }>()
</script>
