<!-- 子图库右栏批量态 (选中 ≥2 时替换详情面板)。 -->
<template>
  <aside class="w-80 shrink-0 border-l border-default overflow-y-auto bg-default">
    <div class="h-full flex flex-col items-center justify-center text-center px-6 py-10">
      <UIcon name="i-tabler-stack-2" class="size-10 text-dimmed mb-3" />
      <p class="text-sm text-toned">{{ t('library.batch.selected_n', { n: count }) }}</p>
      <div class="flex flex-col gap-2 w-full mt-5">
        <UButton size="sm" variant="soft" color="primary" icon="i-tabler-tags" block @click="$emit('batch-add-tags')">
          {{ t('library.batch.add_tags') }}
        </UButton>
        <UButton size="sm" variant="soft" color="error" icon="i-tabler-trash" block @click="$emit('batch-delete')">
          {{ t('library.batch.delete') }}
        </UButton>
        <UButton size="sm" variant="ghost" color="neutral" block @click="$emit('clear')">
          {{ t('library.batch.clear') }}
        </UButton>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

defineProps<{ count: number }>()
defineEmits<{ 'batch-delete': []; 'batch-add-tags': []; clear: [] }>()
</script>
