<template>
  <div class="flex-1 flex flex-col items-center justify-center p-8 text-center">
    <UIcon name="i-tabler-cloud" class="size-12 text-dimmed mb-3" />
    <h3 class="text-sm text-toned font-medium">{{ t('library.online.title') }}</h3>
    <p class="text-xs text-dimmed mt-2 max-w-xs">
      {{ t('library.online.desc') }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
</script>
