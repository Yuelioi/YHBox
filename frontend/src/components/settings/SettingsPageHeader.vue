<template>
  <header class="settings-page-header">
    <div class="settings-page-header__inner">
      <div class="flex min-w-0 items-start gap-3">
        <div
          class="flex size-10 shrink-0 items-center justify-center rounded-lg border border-default bg-elevated/50"
        >
          <UIcon :name="icon" class="size-5 text-primary" aria-hidden="true" />
        </div>
        <div class="min-w-0">
          <p class="mb-1 text-[11px] font-medium tracking-wide text-dimmed">
            {{ t('settingsCenter.eyebrow') }}
          </p>
          <h1 class="text-lg font-semibold leading-tight text-highlighted">{{ title }}</h1>
          <p class="mt-1 max-w-2xl text-xs leading-relaxed text-muted">{{ description }}</p>
        </div>
      </div>
      <SettingsSaveStatus class="shrink-0" />
    </div>
  </header>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import SettingsSaveStatus from './SettingsSaveStatus.vue'

defineProps<{ title: string; description: string; icon: string }>()
const { t } = useI18n()
</script>
