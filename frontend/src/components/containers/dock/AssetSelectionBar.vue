<!-- 批量操作上下文条 (Gmail/GitHub 式): 选中 ≥1 项时冒出, 顶部独占一行 — 已选计数 + 批量动作 + 清除.
     与底部分页解耦, 各占一行不互挤。三个资产面板共用。 -->
<template>
  <div
    v-if="count > 0"
    class="flex items-center gap-2 shrink-0 rounded-md bg-primary/10 ring-1 ring-inset ring-primary/30 px-2.5 py-1.5"
  >
    <span class="text-xs text-toned shrink-0">{{
      t('library.batch.selected_n', { n: count })
    }}</span>
    <UDropdownMenu :items="batchItems">
      <UButton
        size="xs"
        variant="soft"
        color="primary"
        icon="i-tabler-stack-2"
        trailing-icon="i-tabler-chevron-down"
      >
        {{ t('library.batch.menu') }}
      </UButton>
    </UDropdownMenu>
    <UButton
      size="xs"
      variant="ghost"
      color="neutral"
      icon="i-tabler-x"
      class="ml-auto"
      :aria-label="t('library.batch.clear')"
      @click="emit('clear')"
    />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{ count: number; batchItems: any[] }>()
const emit = defineEmits<{ clear: [] }>()
const { t } = useI18n()
</script>
