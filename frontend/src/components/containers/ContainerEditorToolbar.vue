<template>
  <div class="shrink-0 h-11 px-3 border-b border-default flex items-center gap-1 bg-default/60">
    <!-- ====== 左组: 回列表 (仅嵌入态) + sidebar + drag-source openers + undo/redo ====== -->
    <UButton
      v-if="!isStandalone"
      size="xs" variant="ghost" color="neutral"
      icon="i-tabler-arrow-left"
      :title="t('editor.toolbar.back_to_list')"
      @click="$emit('back-to-list')"
    />
    <UButton
      size="xs" variant="ghost" color="neutral"
      :icon="paletteCollapsed ? 'i-tabler-layout-sidebar-left-expand' : 'i-tabler-layout-sidebar-left-collapse'"
      :title="paletteCollapsed ? t('editor.toolbar.palette_expand') : t('editor.toolbar.palette_collapse')"
      @click="$emit('update:paletteCollapsed', !paletteCollapsed)"
    />
    <UButton
      size="sm" variant="ghost" color="neutral"
      icon="i-tabler-grid-dots"
      :title="t('editor.toolbar.node_explorer')"
      @click="$emit('open-node-explorer')"
    />
    <UButton
      size="sm" variant="ghost" color="neutral"
      icon="i-tabler-books"
      :title="t('editor.toolbar.library_explorer')"
      @click="$emit('open-library-explorer')"
    />
    <UButton
      size="sm" variant="ghost" color="neutral"
      icon="i-tabler-arrow-back-up"
      :disabled="!canUndo"
      :title="t('editor.toolbar.undo')"
      @click="$emit('undo')"
    />
    <UButton
      size="sm" variant="ghost" color="neutral"
      icon="i-tabler-arrow-forward-up"
      :disabled="!canRedo"
      :title="t('editor.toolbar.redo')"
      @click="$emit('redo')"
    />

    <div class="w-px h-5 bg-default mx-1" />

    <!-- ====== 中组: 录制 + 折叠子图 + 自动布局 ====== -->
    <template v-if="isRecording">
      <UButton size="sm" color="error" variant="solid" icon="i-tabler-square"
               :title="t('editor.toolbar.stop_record_tip', { hk: hotkeys.keyFor('recording.stop', 'F12') })"
               @click="$emit('stop-record')">{{ t('editor.toolbar.stop_record') }}</UButton>
      <span
        v-if="recordingTargetName"
        class="inline-flex items-center gap-1.5 rounded-md bg-error/15 border border-error/40 px-2 py-0.5 text-[11px] text-error"
        :title="t('editor.toolbar.recording_target_tip', { name: recordingTargetName })"
      >
        <span class="size-1.5 rounded-full bg-error animate-pulse" />
        {{ t('editor.toolbar.recording_target', { name: recordingTargetName }) }}
      </span>
    </template>
    <template v-else-if="countdownSec > 0">
      <UButton size="sm" color="warning" variant="solid" icon="i-tabler-x"
               :title="t('editor.toolbar.cancel_countdown_tip')"
               @click="$emit('cancel-countdown')">{{ t('editor.toolbar.cancel_countdown', { n: countdownSec }) }}</UButton>
    </template>
    <template v-else>
      <UButton size="sm" color="primary" variant="soft" icon="i-tabler-circle-dot"
               :title="t('editor.toolbar.record_precise_tip')"
               @click="$emit('record', 'precise')">{{ t('editor.toolbar.record_precise') }}</UButton>
      <UButton size="sm" color="neutral" variant="subtle" icon="i-tabler-bolt"
               :title="t('editor.toolbar.record_simple_tip')"
               @click="$emit('record', 'simple')">{{ t('editor.toolbar.record_simple') }}</UButton>
    </template>
    <UButton size="sm" variant="soft" color="neutral" icon="i-tabler-package-import"
             :disabled="selectedCount === 0"
             :title="t('editor.toolbar.fold_tip')"
             @click="$emit('fold')">{{ t('editor.toolbar.fold') }}</UButton>
    <UDropdownMenu :items="layoutMenuItems">
      <UButton size="sm" variant="ghost" color="neutral" icon="i-tabler-layout-grid"
               :title="t('editor.toolbar.layout_tip')" />
    </UDropdownMenu>
    <UButton
      size="sm"
      variant="ghost"
      :color="snapEnabled ? 'primary' : 'neutral'"
      :icon="snapEnabled ? 'i-tabler-magnet' : 'i-tabler-magnet-off'"
      :title="snapEnabled ? t('editor.toolbar.snap_on') : t('editor.toolbar.snap_off')"
      @click="$emit('toggle-snap')"
    />

    <div class="flex-1" />

    <!-- ====== 右组: 运行状态 + 主操作 + 折叠 inspector ====== -->
    <div
      v-if="execStoreRunning"
      class="inline-flex items-center gap-2 rounded-md bg-emerald-500/15 border border-emerald-500/40 px-2 py-0.5 text-[11px] text-emerald-300"
    >
      <span class="size-1.5 rounded-full bg-emerald-400 animate-pulse" />
      <span>{{ t('editor.toolbar.running') }}</span>
      <span v-if="runningNodeKind" class="text-emerald-200/80">· {{ runningNodeLabel }}</span>
    </div>
    <UButton v-if="execStoreRunning" size="sm" color="error" variant="solid" icon="i-tabler-square"
             :title="t('editor.toolbar.stop_run_tip', { hk: hotkeys.keyFor('system.execution-stop', 'Ctrl+Shift+F9') })"
             @click="$emit('stop-run')">{{ t('editor.toolbar.stop_run') }}</UButton>

    <UButton size="sm" variant="soft" color="neutral" icon="i-tabler-checks"
             :disabled="dirty"
             :title="dirty ? t('editor.toolbar.validate_dirty_tip') : t('editor.toolbar.validate_tip')"
             @click="$emit('validate')">{{ t('editor.toolbar.validate') }}</UButton>
    <UButton size="sm" variant="soft" color="primary" icon="i-tabler-player-play"
             :disabled="dirty || execStoreRunning"
             :title="dirty ? t('editor.toolbar.try_run_dirty_tip') : execStoreRunning ? t('editor.toolbar.try_run_busy_tip') : t('editor.toolbar.try_run_tip')"
             @click="$emit('try-run')">{{ t('editor.toolbar.try_run') }}</UButton>
    <UButton size="sm" color="primary" icon="i-tabler-check" :disabled="!dirty"
             @click="$emit('save')">{{ t('editor.toolbar.save') }}</UButton>

    <div class="w-px h-5 bg-default mx-1" />

    <!-- ⋯ 更多: 收纳低频操作 (新窗口打开 / 重载); 设置保留在主行 -->
    <UDropdownMenu :items="moreMenuItems">
      <UButton size="sm" variant="ghost" color="neutral" icon="i-tabler-dots"
               :title="t('editor.toolbar.more')" />
    </UDropdownMenu>
    <UButton size="sm" variant="ghost" color="neutral" icon="i-tabler-settings"
             :title="t('editor.toolbar.open_settings')"
             @click="$emit('open-settings')" />

    <UButton size="xs" variant="ghost" color="neutral"
             :icon="inspectorCollapsed ? 'i-tabler-layout-sidebar-right-expand' : 'i-tabler-layout-sidebar-right-collapse'"
             :title="inspectorCollapsed ? t('editor.toolbar.inspector_expand') : t('editor.toolbar.inspector_collapse')"
             @click="$emit('update:inspectorCollapsed', !inspectorCollapsed)" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useHotkeysStore } from '@/stores/hotkeys'

const { t } = useI18n()
const hotkeys = useHotkeysStore()

const props = defineProps<{
  paletteCollapsed: boolean
  inspectorCollapsed: boolean
  // recording 三态:  isRecording (后端真的在录) / countdownSec>0 (倒计时中) / 都不是 (空闲)
  isRecording: boolean
  // A4: 录制目标容器名 (录制态显示绑定指示器, 空则不显示)
  recordingTargetName?: string
  countdownSec: number
  selectedCount: number
  execStoreRunning: boolean
  runningNodeKind: string | undefined
  runningNodeLabel: string
  dirty: boolean
  canUndo?: boolean
  canRedo?: boolean
  snapEnabled?: boolean
  isStandalone?: boolean
}>()

const emit = defineEmits<{
  'update:paletteCollapsed': [v: boolean]
  'update:inspectorCollapsed': [v: boolean]
  // 'record' 带 mode 参数: 'precise' | 'simple'
  'record': [mode: 'precise' | 'simple']
  'stop-record': []
  'cancel-countdown': []
  'fold': []
  'try-run': []
  'stop-run': []
  'save': []
  'reload': []
  'validate': []
  'auto-layout': [direction: 'LR' | 'TB']
  'align-selected': [
    mode: 'left' | 'right' | 'top' | 'bottom' | 'center-h' | 'center-v' | 'h-equal' | 'v-equal',
  ]
  // 工具栏按钮 emits (实际 modal 在父 ContainerEditorView 里挂):
  'open-node-explorer': []
  'open-library-explorer': []
  'open-settings': []
  'undo': []
  'redo': []
  'toggle-snap': []
  'open-new-window': []
  'back-to-list': []
}>()

// ⋯ 更多菜单: 低频操作 (新窗口打开仅嵌入态有; 重载两态都有). 设置保留在主行, 不进菜单.
const moreMenuItems = computed(() => {
  const items: { label: string; icon: string; onSelect: () => void }[] = []
  if (!props.isStandalone) {
    items.push({
      label: t('editor.toolbar.open_new_window'),
      icon: 'i-tabler-external-link',
      onSelect: () => emit('open-new-window'),
    })
  }
  items.push({
    label: t('editor.toolbar.reload'),
    icon: 'i-tabler-refresh',
    onSelect: () => emit('reload'),
  })
  return [items]
})

const layoutMenuItems = computed(() => [
  [
    {
      label: t('editor.layout.auto_lr'),
      icon: 'i-tabler-layout-board-split',
      onSelect: () => emit('auto-layout', 'LR'),
    },
    {
      label: t('editor.layout.auto_tb'),
      icon: 'i-tabler-layout-rows',
      onSelect: () => emit('auto-layout', 'TB'),
    },
  ],
  [
    {
      label: t('editor.layout.align_left'),
      icon: 'i-tabler-align-box-left-middle',
      disabled: props.selectedCount < 2,
      onSelect: () => emit('align-selected', 'left'),
    },
    {
      label: t('editor.layout.align_right'),
      icon: 'i-tabler-align-box-right-middle',
      disabled: props.selectedCount < 2,
      onSelect: () => emit('align-selected', 'right'),
    },
    {
      label: t('editor.layout.align_top'),
      icon: 'i-tabler-align-box-top-center',
      disabled: props.selectedCount < 2,
      onSelect: () => emit('align-selected', 'top'),
    },
    {
      label: t('editor.layout.align_bottom'),
      icon: 'i-tabler-align-box-bottom-center',
      disabled: props.selectedCount < 2,
      onSelect: () => emit('align-selected', 'bottom'),
    },
    {
      label: t('editor.layout.center_h'),
      icon: 'i-tabler-align-center-horizontal',
      disabled: props.selectedCount < 2,
      onSelect: () => emit('align-selected', 'center-h'),
    },
    {
      label: t('editor.layout.center_v'),
      icon: 'i-tabler-align-center-vertical',
      disabled: props.selectedCount < 2,
      onSelect: () => emit('align-selected', 'center-v'),
    },
  ],
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
