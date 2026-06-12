<template>
  <div class="flex items-center gap-1.5 text-xs min-w-0">
    <!-- 根 = 主图: 可点回主图 (goto -1 → editorStore 清空 path)。在子图层级才需要可点; 已在主图时纯展示。 -->
    <UButton
      v-if="editorPath.length > 0"
      size="xs"
      variant="ghost"
      color="neutral"
      class="!px-1.5 !py-0 !h-auto text-toned truncate max-w-[180px]"
      @click="$emit('goto', -1)"
    >{{ rootLabel ?? t('editor.breadcrumb.root_fallback') }}</UButton>
    <span v-else class="text-toned truncate max-w-[180px]">{{ rootLabel ?? t('editor.breadcrumb.root_fallback') }}</span>
    <template v-for="(sgID, idx) in editorPath" :key="sgID">
      <UIcon name="i-tabler-chevron-right" class="size-3 text-dimmed" />
      <UButton
        size="xs"
        variant="ghost"
        color="neutral"
        class="!px-1.5 !py-0 !h-auto"
        @click="$emit('goto', idx)"
      >{{ sgLabelFn(sgID) }}</UButton>
    </template>
    <!-- 当前层级节点数 (Subgraph 算 1, 主图节点总数 / 子图内部节点总数) -->
    <span
      v-if="activeNodeCount !== null"
      class="ml-2 text-[10px] text-dimmed inline-flex items-center gap-1"
      :title="editorPath.length === 0 ? t('editor.breadcrumb.tooltip_main') : t('editor.breadcrumb.tooltip_subgraph')"
    >
      <UIcon name="i-tabler-cpu" class="size-3" />
      {{ t('editor.breadcrumb.node_count', { n: activeNodeCount }) }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

defineProps<{
  rootLabel: string | undefined
  editorPath: readonly string[]
  sgLabelFn: (id: string) => string
  activeNodeCount: number | null
}>()

defineEmits<{
  'goto': [idx: number] // idx -1 = 回主图根; >=0 = 跳到该层子图
}>()
</script>
