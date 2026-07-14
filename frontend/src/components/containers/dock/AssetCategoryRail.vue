<template>
  <aside class="w-44 shrink-0 overflow-y-auto border-r border-default bg-elevated/10 p-2.5">
    <div class="mb-2 flex items-center gap-2 px-2 text-xs font-medium text-muted">
      <UIcon name="i-tabler-category" class="size-3.5" />
      <span>{{ t('assetBrowser.categories') }}</span>
    </div>
    <div class="space-y-0.5">
      <UButton
        v-for="item in items"
        :key="item.id"
        block
        size="xs"
        color="neutral"
        :variant="model === item.id ? 'soft' : 'ghost'"
        class="justify-start rounded-md"
        :class="model === item.id ? 'text-primary' : 'text-muted'"
        :aria-pressed="model === item.id"
        @click="model = item.id"
      >
        <span class="min-w-0 flex-1 truncate text-left">{{ item.label }}</span>
        <span v-if="item.count !== undefined" class="text-[11px] text-dimmed">{{
          item.count
        }}</span>
      </UButton>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{ items: Array<{ id: string; label: string; count?: number }> }>()
const model = defineModel<string>({ required: true })
const { t } = useI18n()
</script>
