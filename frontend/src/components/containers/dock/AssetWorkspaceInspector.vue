<template>
  <aside
    class="asset-workspace-inspector absolute inset-0 z-20 min-h-0 flex-col bg-default"
    :aria-label="title"
  >
    <header class="flex shrink-0 items-center gap-2 border-b border-default px-3 py-2">
      <UButton
        size="xs"
        variant="ghost"
        color="neutral"
        icon="i-tabler-arrow-left"
        @click="emit('close')"
      >
        {{ t('common.back') }}
      </UButton>
      <span class="min-w-0 flex-1 truncate text-sm font-medium text-highlighted">{{ title }}</span>
    </header>
    <div class="min-h-0 flex-1 overflow-y-auto">
      <slot />
    </div>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{ title: string }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
</script>

<style scoped>
.asset-workspace-inspector {
  display: none;
}

@container (width < 1040px) {
  .asset-workspace-inspector {
    display: flex;
  }
}
</style>
