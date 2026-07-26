<template>
  <div class="settings-row">
    <div class="min-w-0">
      <div class="flex flex-wrap items-center gap-2">
        <div class="text-sm font-medium text-default">{{ label }}</div>
        <slot name="meta" />
      </div>
      <p v-if="hint" class="mt-1 max-w-2xl text-xs leading-relaxed text-dimmed">{{ hint }}</p>
    </div>
    <div class="settings-row__control">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ label: string; hint?: string }>()
</script>
