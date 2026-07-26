<template>
  <article class="rounded-lg border border-default bg-elevated/10 p-3">
    <div class="flex items-center gap-2">
      <UIcon :name="icon" class="size-4 text-primary" />
      <h5 class="text-xs font-semibold text-highlighted">{{ title }}</h5>
      <UBadge color="neutral" variant="soft" size="xs">{{ items.length }}</UBadge>
    </div>
    <ul v-if="items.length" class="mt-2 space-y-1 text-[11px] text-muted">
      <li v-for="item in items" :key="item" class="break-all">{{ item }}</li>
    </ul>
    <p v-else class="mt-2 text-[11px] text-dimmed">—</p>
  </article>
</template>

<script setup lang="ts">
defineProps<{ title: string; icon: string; items: string[] }>()
</script>
