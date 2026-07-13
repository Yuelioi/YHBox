<!-- 画布浮动上下文条 (Figma 风)：选中节点时画布顶部居中浮现，放选区操作
     (折叠 / 对齐 / 分布 / 删除)。不选时父级 v-if 整条隐藏，画布最干净。 -->
<template>
  <div data-testid="canvas-context-bar" class="canvas-context-bar">
    <span class="context-selection context-selection--full">{{
      t('editor.canvas.context.selected', { n: selectedCount })
    }}</span>
    <span class="context-selection context-selection--compact">{{ selectedCount }}</span>
    <div class="context-divider" />

    <UButton
      size="xs"
      variant="ghost"
      color="neutral"
      icon="i-tabler-package-import"
      :title="t('editor.toolbar.fold_tip')"
      :aria-label="t('editor.toolbar.fold')"
      @click="$emit('fold')"
      ><span class="context-action-label">{{ t('editor.toolbar.fold') }}</span></UButton
    >

    <UDropdownMenu :items="alignMenuItems">
      <UButton
        size="xs"
        variant="ghost"
        color="neutral"
        icon="i-tabler-layout-align-center"
        :disabled="selectedCount < 2"
        :title="t('editor.canvas.context.align')"
        :aria-label="t('editor.canvas.context.align')"
        ><span class="context-action-label">{{ t('editor.canvas.context.align') }}</span></UButton
      >
    </UDropdownMenu>

    <UDropdownMenu :items="distributeMenuItems">
      <UButton
        size="xs"
        variant="ghost"
        color="neutral"
        icon="i-tabler-arrows-horizontal"
        :disabled="selectedCount < 3"
        :title="t('editor.canvas.context.distribute')"
        :aria-label="t('editor.canvas.context.distribute')"
        ><span class="context-action-label">{{
          t('editor.canvas.context.distribute')
        }}</span></UButton
      >
    </UDropdownMenu>

    <div class="context-divider" />
    <UButton
      size="xs"
      variant="ghost"
      color="error"
      icon="i-tabler-trash"
      :title="t('editor.canvas.context.delete')"
      :aria-label="t('editor.canvas.context.delete')"
      @click="$emit('delete-selected')"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<{ selectedCount: number }>()

type AlignMode =
  | 'left'
  | 'right'
  | 'top'
  | 'bottom'
  | 'center-h'
  | 'center-v'
  | 'h-equal'
  | 'v-equal'

const emit = defineEmits<{
  fold: []
  'align-selected': [mode: AlignMode]
  'delete-selected': []
}>()

const alignMenuItems = computed(() => [
  [
    {
      label: t('editor.layout.align_left'),
      icon: 'i-tabler-align-box-left-middle',
      onSelect: () => emit('align-selected', 'left'),
    },
    {
      label: t('editor.layout.align_right'),
      icon: 'i-tabler-align-box-right-middle',
      onSelect: () => emit('align-selected', 'right'),
    },
    {
      label: t('editor.layout.align_top'),
      icon: 'i-tabler-align-box-top-center',
      onSelect: () => emit('align-selected', 'top'),
    },
    {
      label: t('editor.layout.align_bottom'),
      icon: 'i-tabler-align-box-bottom-center',
      onSelect: () => emit('align-selected', 'bottom'),
    },
    {
      label: t('editor.layout.center_h'),
      icon: 'i-tabler-layout-align-middle',
      onSelect: () => emit('align-selected', 'center-h'),
    },
    {
      label: t('editor.layout.center_v'),
      icon: 'i-tabler-layout-align-center',
      onSelect: () => emit('align-selected', 'center-v'),
    },
  ],
])

const distributeMenuItems = computed(() => [
  [
    {
      label: t('editor.layout.dist_h'),
      icon: 'i-tabler-arrows-horizontal',
      disabled: props.selectedCount < 3,
      onSelect: () => emit('align-selected', 'h-equal'),
    },
    {
      label: t('editor.layout.dist_v'),
      icon: 'i-tabler-arrows-vertical',
      disabled: props.selectedCount < 3,
      onSelect: () => emit('align-selected', 'v-equal'),
    },
  ],
])
</script>

<style scoped>
.canvas-context-bar {
  position: absolute;
  top: 12px;
  left: 50%;
  z-index: 20;
  display: inline-flex;
  max-width: calc(100% - 24px);
  transform: translateX(-50%);
  align-items: center;
  gap: 4px;
  padding: 5px 7px;
  overflow: hidden;
  border: 1px solid var(--ui-border-accented);
  border-radius: 8px;
  background: color-mix(in oklch, var(--ui-bg-elevated) 96%, var(--ui-primary));
  box-shadow: 0 10px 28px color-mix(in oklch, var(--ui-bg) 76%, transparent);
  white-space: nowrap;
}

.context-selection {
  padding-inline: 6px;
  color: var(--ui-text-muted);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.context-selection--compact {
  display: none;
  color: var(--ui-primary);
  font-weight: 600;
}

.context-divider {
  width: 1px;
  height: 16px;
  margin-inline: 2px;
  flex: none;
  background: var(--ui-border);
}

@container editor-canvas (max-width: 540px) {
  .context-action-label,
  .context-selection--full {
    display: none;
  }

  .context-selection--compact {
    display: inline;
  }
}
</style>
