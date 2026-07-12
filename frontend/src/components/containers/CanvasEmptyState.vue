<!-- 画布空态: activeGraph 无节点时居中显, 取代原 Inspector 常占一栏的「快捷开始」。 -->
<template>
  <div class="absolute inset-0 z-10 flex items-center justify-center pointer-events-none">
    <div class="overlay-surface rounded-xl px-6 py-5 max-w-xs space-y-3">
      <p class="text-sm font-medium text-highlighted">
        {{ t('editor.inspector.empty.quick_start') }}
      </p>
      <ul class="space-y-2 text-[12px] text-toned">
        <li class="flex items-center gap-2">
          <kbd class="px-1.5 py-0.5 bg-elevated rounded border border-default text-[10px]">Tab</kbd>
          <span>{{ t('editor.inspector.empty.tab_explorer') }}</span>
        </li>
        <li class="flex items-center gap-2">
          <UIcon name="i-tabler-mouse" class="size-3.5" />
          <span>{{ t('editor.inspector.empty.right_click_canvas') }}</span>
        </li>
        <li class="flex items-center gap-2">
          <kbd class="px-1.5 py-0.5 bg-elevated rounded border border-default text-[10px]"
            >Ctrl+K</kbd
          >
          <span>{{ t('editor.inspector.empty.command_palette') }}</span>
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
const { t } = useI18n()
</script>
