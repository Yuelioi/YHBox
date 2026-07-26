<template>
  <UBadge color="warning" variant="subtle" size="xs" icon="i-tabler-reload">
    {{ t('settingsCenter.restart_required') }}
  </UBadge>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
</script>
