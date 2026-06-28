<template>
  <!-- 独立工具窗统一外壳 (录屏/截图/鼠标检测/校准/启动器共用)。
       frameless 窗的 chrome: 主题底 + 可拖标题栏 (图标 + 标题 + actions + 关闭) + body slot。 -->
  <div class="h-screen w-screen bg-default text-default flex flex-col select-none">
    <header
      class="shrink-0 flex items-center gap-2 border-b border-default"
      :class="dense ? 'px-2 py-1' : 'px-3 py-2'"
      style="--wails-draggable: drag"
    >
      <UIcon v-if="icon" :name="icon" class="size-4 shrink-0" :class="accentClass" />
      <h3 v-if="title" class="text-xs font-medium text-highlighted truncate">{{ title }}</h3>
      <UIcon v-if="!icon && !title" name="i-tabler-grip-horizontal" class="size-3 text-dimmed shrink-0" />
      <span class="ml-auto" />
      <div class="flex items-center gap-0.5" style="--wails-draggable: no-drag">
        <slot name="actions" />
        <UButton
          v-if="!noClose"
          size="xs" variant="ghost" color="neutral" icon="i-tabler-x"
          :title="closeTitle" @click="emit('close')"
        />
      </div>
    </header>
    <div class="flex-1 min-h-0 flex flex-col">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = withDefaults(
  defineProps<{
    title?: string
    icon?: string
    accent?: 'primary' | 'error' | 'success' | 'warning' | 'neutral'
    dense?: boolean // 紧凑标题栏 (工具条用)
    noClose?: boolean
    closeTitle?: string
  }>(),
  { accent: 'primary' },
)
const emit = defineEmits<{ (e: 'close'): void }>()
const closeTitle = computed(() => props.closeTitle ?? t('hudShell.close'))

const accentClass = computed(
  () =>
    ({
      primary: 'text-primary',
      error: 'text-error',
      success: 'text-success',
      warning: 'text-warning',
      neutral: 'text-dimmed',
    })[props.accent],
)
</script>
