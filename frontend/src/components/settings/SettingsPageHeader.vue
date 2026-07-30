<template>
  <header class="settings-page-header">
    <div class="settings-page-header__inner">
      <div class="flex min-w-0 items-start gap-3">
        <div class="settings-page-header__icon">
          <UIcon :name="icon" class="size-5" aria-hidden="true" />
        </div>
        <div class="min-w-0">
          <h1 class="settings-page-header__title">{{ title }}</h1>
          <p class="settings-page-header__description">{{ description }}</p>
        </div>
      </div>
      <SettingsSaveStatus class="shrink-0" />
    </div>
  </header>
</template>

<script setup lang="ts">
import SettingsSaveStatus from './SettingsSaveStatus.vue'

defineProps<{ title: string; description: string; icon: string }>()
</script>
