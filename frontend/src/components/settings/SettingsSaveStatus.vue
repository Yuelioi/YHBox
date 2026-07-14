<template>
  <div
    class="inline-flex min-h-8 items-center gap-2 rounded-md px-2.5 text-xs"
    :class="statusClass"
    role="status"
    aria-live="polite"
    aria-atomic="true"
  >
    <UIcon
      :name="statusIcon"
      class="size-3.5 shrink-0"
      :class="saveState === 'saving' ? 'animate-spin motion-reduce:animate-none' : ''"
      aria-hidden="true"
    />
    <span>{{ statusLabel }}</span>
    <UButton
      v-if="saveState === 'error'"
      size="xs"
      variant="link"
      color="error"
      class="-mr-1 px-1"
      :label="t('common.retry')"
      @click="settingsStore.retryLastPatch"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '@/stores/settings'

const { t } = useI18n()
const settingsStore = useSettingsStore()
const saveState = computed(() => settingsStore.saveState)

const statusLabel = computed(() => {
  if (saveState.value === 'saving') return t('settingsCenter.save.saving')
  if (saveState.value === 'saved') return t('settingsCenter.save.saved')
  if (saveState.value === 'error') return t('settingsCenter.save.failed')
  return t('settingsCenter.save.automatic')
})

const statusIcon = computed(() => {
  if (saveState.value === 'saving') return 'i-tabler-loader-2'
  if (saveState.value === 'saved') return 'i-tabler-cloud-check'
  if (saveState.value === 'error') return 'i-tabler-alert-circle'
  return 'i-tabler-device-floppy'
})

const statusClass = computed(() => {
  if (saveState.value === 'saved') return 'bg-success/10 text-success'
  if (saveState.value === 'error') return 'bg-error/10 text-error'
  if (saveState.value === 'saving') return 'bg-primary/10 text-primary'
  return 'bg-elevated/50 text-dimmed'
})
</script>
