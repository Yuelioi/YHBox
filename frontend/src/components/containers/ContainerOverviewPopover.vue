<!-- 容器概览 popover: 节点/变量/子图数 + 热键 + 帮助入口。挂 toolbar 左区面包屑旁。 -->
<template>
  <UPopover>
    <UButton
      size="xs"
      variant="ghost"
      color="neutral"
      icon="i-tabler-info-circle"
      :title="t('editor.overview.title')"
    />
    <template #content>
      <div class="p-3 w-56 space-y-3">
        <p class="text-xs font-medium text-highlighted">{{ t('editor.overview.title') }}</p>
        <div class="flex flex-wrap gap-1.5">
          <span class="px-1.5 py-0.5 rounded bg-elevated/60 border border-default/60 text-[11px]">
            {{ t('editor.inspector.empty.stats_nodes', { n: nodeCount }) }}
          </span>
          <span class="px-1.5 py-0.5 rounded bg-elevated/60 border border-default/60 text-[11px]">
            {{ t('editor.inspector.empty.stats_vars', { n: varCount }) }}
          </span>
          <span class="px-1.5 py-0.5 rounded bg-elevated/60 border border-default/60 text-[11px]">
            {{ t('editor.inspector.empty.stats_subgraphs', { n: subgraphCount }) }}
          </span>
        </div>
        <div class="flex items-center gap-2 text-[11px]">
          <span class="text-dimmed">{{ t('editor.inspector.empty.hotkey_label') }}</span>
          <kbd
            v-if="hotkey"
            class="px-1.5 py-0.5 bg-elevated rounded border border-default text-[10px]"
            >{{ hotkey }}</kbd
          >
          <span v-else class="text-dimmed">{{ t('editor.inspector.empty.hotkey_none') }}</span>
        </div>
        <button
          type="button"
          class="flex items-center gap-2 w-full px-3 py-2 rounded-md bg-elevated/30 border border-default/40 text-left hover:bg-elevated/50 transition-colors"
          @click="$emit('open-help')"
        >
          <UIcon name="i-tabler-help-circle" class="size-3.5 text-primary shrink-0" />
          <span class="text-[12px] font-medium text-toned flex-1">{{
            t('editor.inspector.empty.open_help')
          }}</span>
          <UIcon name="i-tabler-chevron-right" class="size-3.5 text-dimmed shrink-0" />
        </button>
      </div>
    </template>
  </UPopover>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
const { t } = useI18n()
defineProps<{ nodeCount: number; varCount: number; subgraphCount: number; hotkey: string }>()
defineEmits<{ 'open-help': [] }>()
</script>
