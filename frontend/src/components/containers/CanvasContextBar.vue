<!-- 画布浮动上下文条 (Figma 风)：选中节点时画布顶部居中浮现，放选区操作
     (折叠 / 对齐 / 分布 / 删除)。不选时父级 v-if 整条隐藏，画布最干净。 -->
<template>
  <div
    class="absolute top-3 left-1/2 -translate-x-1/2 z-20 inline-flex items-center gap-1 rounded-full bg-default/95 border border-primary/50 shadow-lg px-2 py-1 backdrop-blur"
  >
    <span class="text-[11px] text-dimmed px-1.5">{{ t('editor.canvas.context.selected', { n: selectedCount }) }}</span>
    <div class="w-px h-4 bg-default mx-0.5" />

    <UButton
      size="xs" variant="ghost" color="neutral" icon="i-tabler-package-import"
      :title="t('editor.toolbar.fold_tip')"
      @click="$emit('fold')"
    >{{ t('editor.toolbar.fold') }}</UButton>

    <UDropdownMenu :items="alignMenuItems">
      <UButton
        size="xs" variant="ghost" color="neutral" icon="i-tabler-layout-align-center"
        :disabled="selectedCount < 2"
        :title="t('editor.canvas.context.align')"
      >{{ t('editor.canvas.context.align') }}</UButton>
    </UDropdownMenu>

    <UDropdownMenu :items="distributeMenuItems">
      <UButton
        size="xs" variant="ghost" color="neutral" icon="i-tabler-arrows-horizontal"
        :disabled="selectedCount < 3"
        :title="t('editor.canvas.context.distribute')"
      >{{ t('editor.canvas.context.distribute') }}</UButton>
    </UDropdownMenu>

    <div class="w-px h-4 bg-default mx-0.5" />
    <UButton
      size="xs" variant="ghost" color="error" icon="i-tabler-trash"
      :title="t('editor.canvas.context.delete')"
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
  | 'left' | 'right' | 'top' | 'bottom' | 'center-h' | 'center-v' | 'h-equal' | 'v-equal'

const emit = defineEmits<{
  fold: []
  'align-selected': [mode: AlignMode]
  'delete-selected': []
}>()

const alignMenuItems = computed(() => [[
  { label: t('editor.layout.align_left'), icon: 'i-tabler-align-box-left-middle', onSelect: () => emit('align-selected', 'left') },
  { label: t('editor.layout.align_right'), icon: 'i-tabler-align-box-right-middle', onSelect: () => emit('align-selected', 'right') },
  { label: t('editor.layout.align_top'), icon: 'i-tabler-align-box-top-center', onSelect: () => emit('align-selected', 'top') },
  { label: t('editor.layout.align_bottom'), icon: 'i-tabler-align-box-bottom-center', onSelect: () => emit('align-selected', 'bottom') },
  { label: t('editor.layout.center_h'), icon: 'i-tabler-layout-align-middle', onSelect: () => emit('align-selected', 'center-h') },
  { label: t('editor.layout.center_v'), icon: 'i-tabler-layout-align-center', onSelect: () => emit('align-selected', 'center-v') },
]])

const distributeMenuItems = computed(() => [[
  { label: t('editor.layout.dist_h'), icon: 'i-tabler-arrows-horizontal', disabled: props.selectedCount < 3, onSelect: () => emit('align-selected', 'h-equal') },
  { label: t('editor.layout.dist_v'), icon: 'i-tabler-arrows-vertical', disabled: props.selectedCount < 3, onSelect: () => emit('align-selected', 'v-equal') },
]])
</script>
