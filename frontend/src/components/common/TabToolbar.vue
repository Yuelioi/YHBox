<!--
TabToolbar 公共组件：view 顶部 tabs + actions slot 封装。
用于 ContainersView / LibraryView 顶部统一布局。
-->
<template>
  <div class="shrink-0 h-12 px-4 flex items-center gap-3 border-b border-default">
    <UTabs :items="tabs" v-model="activeTab" size="sm" :ui="{ list: 'h-9' }" />
    <div class="grow" />
    <slot name="actions" />
  </div>
</template>

<script setup lang="ts">
defineProps<{ tabs: { label: string; value: string }[] }>()
const activeTab = defineModel<string>('activeTab', { required: true })
</script>
