<template>
  <section class="settings-section">
    <div class="settings-section__heading">
      <span v-if="icon" class="settings-section__icon">
        <UIcon :name="icon" class="size-4" aria-hidden="true" />
      </span>
      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-2">
          <h2 class="settings-section__title">{{ title }}</h2>
          <slot name="badge" />
        </div>
        <p v-if="description" class="settings-section__description">
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
