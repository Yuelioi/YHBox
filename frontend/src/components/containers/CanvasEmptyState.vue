<!-- 画布空态: activeGraph 无节点时提供明确的首个动作，并保留次要快捷键提示。 -->
<template>
  <div class="pointer-events-none absolute inset-0 z-10 flex items-center justify-center px-6">
    <section class="canvas-empty-card pointer-events-auto w-full max-w-sm px-6 py-5">
      <div class="mb-4 flex items-start gap-3">
        <div class="flex size-9 shrink-0 items-center justify-center rounded-lg bg-primary/10">
          <UIcon name="i-tabler-route" class="size-5 text-primary" />
        </div>
        <div class="min-w-0">
          <h2 class="text-[15px] font-semibold text-highlighted">
            {{ t('editor.inspector.empty.canvas_title') }}
          </h2>
          <p class="mt-1 text-[13px] leading-5 text-muted">
            {{ t('editor.inspector.empty.canvas_desc') }}
          </p>
        </div>
      </div>

      <div class="flex flex-wrap gap-2">
        <UButton icon="i-tabler-plus" color="primary" @click="emit('open-nodes')">
          {{ t('editor.inspector.empty.add_first_node') }}
        </UButton>
        <UButton
          icon="i-tabler-player-record"
          color="neutral"
          variant="soft"
          @click="emit('record')"
        >
          {{ t('editor.inspector.empty.record_actions') }}
        </UButton>
      </div>

      <div class="mt-5 border-t border-default pt-4">
        <p class="mb-2 text-xs font-medium text-muted">
          {{ t('editor.inspector.empty.shortcuts_label') }}
        </p>
        <ul class="space-y-2 text-xs text-toned">
          <li class="flex items-center gap-2">
            <kbd class="rounded border border-default bg-elevated px-1.5 py-0.5 text-[11px]"
              >Tab</kbd
            >
            <span>{{ t('editor.inspector.empty.tab_explorer') }}</span>
          </li>
          <li class="flex items-center gap-2">
            <UIcon name="i-tabler-mouse" class="size-3.5" />
            <span>{{ t('editor.inspector.empty.right_click_canvas') }}</span>
          </li>
          <li class="flex items-center gap-2">
            <kbd class="rounded border border-default bg-elevated px-1.5 py-0.5 text-[11px]"
              >Ctrl+K</kbd
            >
            <span>{{ t('editor.inspector.empty.command_palette') }}</span>
          </li>
        </ul>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const emit = defineEmits<{
  'open-nodes': []
  record: []
}>()
const { t } = useI18n()
</script>

<style scoped>
.canvas-empty-card {
  border: 1px solid var(--ui-border-accented);
  border-radius: 10px;
  background: color-mix(in oklab, var(--ui-bg-elevated) 88%, var(--ui-bg));
  box-shadow: 0 12px 28px rgba(0, 0, 0, 0.28);
}
</style>
