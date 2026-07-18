<template>
  <div
    data-testid="workflow-selection-toolbar"
    class="absolute left-1/2 top-3 z-20 flex -translate-x-1/2 items-center gap-1 rounded-xl border border-default bg-default/95 p-1.5 shadow-xl"
  >
    <span class="shrink-0 whitespace-nowrap px-2 text-[11px] font-medium text-muted">
      {{ t('workflow.selection.count', { count }) }}
    </span>
    <div class="h-5 w-px bg-default ring-1 ring-default" />
    <UButton
      icon="i-tabler-copy"
      color="neutral"
      variant="ghost"
      size="xs"
      :aria-label="t('workflow.selection.copy')"
      @click="emit('copy')"
    />
    <UButton
      icon="i-tabler-cut"
      color="neutral"
      variant="ghost"
      size="xs"
      :aria-label="t('workflow.selection.cut')"
      @click="emit('cut')"
    />
    <UButton
      icon="i-tabler-copy-plus"
      color="neutral"
      variant="ghost"
      size="xs"
      :aria-label="t('workflow.selection.duplicate')"
      @click="emit('duplicate')"
    />
    <UButton
      icon="i-tabler-folders"
      color="neutral"
      variant="ghost"
      size="xs"
      :label="t('workflow.selection.collapse')"
      @click="emit('collapse')"
    />
    <UDropdownMenu :items="layoutItems">
      <UButton
        icon="i-tabler-layout-align-middle"
        color="neutral"
        variant="ghost"
        size="xs"
        :label="t('workflow.selection.arrange')"
      />
    </UDropdownMenu>
    <UButton
      data-testid="workflow-layout-lr"
      icon="i-tabler-layout-board-split"
      color="neutral"
      variant="ghost"
      size="xs"
      :loading="layouting"
      :label="t('workflow.selection.layout_lr')"
      @click="emit('auto-layout', 'LR')"
    />
    <UButton
      data-testid="workflow-layout-tb"
      icon="i-tabler-layout-navbar-collapse"
      color="neutral"
      variant="ghost"
      size="xs"
      :loading="layouting"
      :aria-label="t('workflow.selection.layout_tb')"
      @click="emit('auto-layout', 'TB')"
    />
    <UButton
      icon="i-tabler-trash"
      color="error"
      variant="ghost"
      size="xs"
      :aria-label="t('workflow.selection.remove')"
      @click="emit('remove')"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AlignMode, DistributeMode } from './workflowLayout'

const props = defineProps<{ count: number; layouting: boolean }>()
const emit = defineEmits<{
  align: [mode: AlignMode]
  distribute: [mode: DistributeMode]
  'auto-layout': [direction: 'LR' | 'TB']
  copy: []
  cut: []
  duplicate: []
  collapse: []
  remove: []
}>()
const { t } = useI18n()

const layoutItems = computed(() => [
  [
    action('workflow.selection.align_left', 'i-tabler-layout-align-left', () =>
      emit('align', 'left'),
    ),
    action('workflow.selection.align_right', 'i-tabler-layout-align-right', () =>
      emit('align', 'right'),
    ),
    action('workflow.selection.align_top', 'i-tabler-layout-align-top', () => emit('align', 'top')),
    action('workflow.selection.align_bottom', 'i-tabler-layout-align-bottom', () =>
      emit('align', 'bottom'),
    ),
    action('workflow.selection.align_horizontal', 'i-tabler-layout-align-middle', () =>
      emit('align', 'horizontal-center'),
    ),
    action('workflow.selection.align_vertical', 'i-tabler-layout-align-center', () =>
      emit('align', 'vertical-center'),
    ),
  ],
  [
    action(
      'workflow.selection.distribute_horizontal',
      'i-tabler-spacing-horizontal',
      () => emit('distribute', 'horizontal'),
      props.count < 3,
    ),
    action(
      'workflow.selection.distribute_vertical',
      'i-tabler-spacing-vertical',
      () => emit('distribute', 'vertical'),
      props.count < 3,
    ),
  ],
])

function action(key: string, icon: string, onSelect: () => void, disabled = props.count < 2) {
  return { label: t(key), icon, disabled, onSelect }
}
</script>
