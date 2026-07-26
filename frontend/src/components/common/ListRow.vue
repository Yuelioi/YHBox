<!-- 列表行: icon / 默认 / trailing 三槽。静止扁平; hover 套 raised 轻浮起。active=true 常驻浮起。 -->
<template>
  <div
    class="flex items-center gap-2.5 rounded-lg px-3 py-2 transition-colors duration-150"
    :class="active ? 'raised-surface' : 'border border-transparent hover:raised-surface'"
  >
    <slot name="icon" />
    <div class="min-w-0 flex-1">
      <slot />
    </div>
    <slot name="trailing" />
  </div>
</template>

<script setup lang="ts">
defineProps<{ active?: boolean }>()
</script>
