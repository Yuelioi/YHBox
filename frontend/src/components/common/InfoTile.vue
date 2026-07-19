<template>
  <div class="flex min-w-0 items-center gap-2 rounded-lg bg-elevated/45 px-3 py-2">
    <UIcon :name="icon" class="size-4 shrink-0 text-primary" />
    <div class="min-w-0">
      <p class="truncate text-[10px] text-dimmed">{{ label }}</p>
      <p class="truncate text-xs font-medium text-toned"><slot /></p>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ icon: string; label: string }>()
</script>
