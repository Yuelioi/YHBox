<template>
  <footer class="flex shrink-0 items-center justify-between gap-3 border-t border-default pt-2">
    <span class="shrink-0 text-xs text-dimmed">{{ t('library.toolbar.total', { n: total }) }}</span>
    <UPagination
      v-if="totalPages > 1"
      v-model:page="page"
      :total="total"
      :items-per-page="itemsPerPage"
      :sibling-count="1"
      size="xs"
      :aria-label="t('assetBrowser.pagination')"
    />
  </footer>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{ total: number; totalPages: number; itemsPerPage: number }>()
const page = defineModel<number>('page', { required: true })
const { t } = useI18n()
</script>
