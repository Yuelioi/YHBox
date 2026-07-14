<template>
  <section class="settings-section">
    <div class="flex items-start gap-3">
      <UIcon
        v-if="icon"
        :name="icon"
        class="mt-0.5 size-4 shrink-0 text-dimmed"
        aria-hidden="true"
      />
      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-2">
          <h2 class="text-sm font-semibold text-highlighted">{{ title }}</h2>
          <slot name="badge" />
        </div>
        <p v-if="description" class="mt-1 max-w-2xl text-xs leading-relaxed text-dimmed">
          {{ description }}
        </p>
      </div>
      <slot name="actions" />
    </div>
    <slot />
  </section>
</template>

<script setup lang="ts">
defineProps<{ title: string; description?: string; icon?: string }>()
</script>
