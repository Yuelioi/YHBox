<template>
  <!-- C2 单行布局：左 [导航 + 面包屑身份] · 中 [撤销/重做 · 加内容 · 录制] · 右 [运行/主操作 · ⋯/设置 · 折叠属性] -->
  <div class="shrink-0 h-11 px-3 border-b border-default flex items-center gap-1 bg-default/60">
    <!-- ====== 左: 回列表(嵌入态) + palette 折叠 + 面包屑身份 ====== -->
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
    <ContainerEditorBreadcrumb
      class="ml-1 min-w-0"
      :root-label="rootLabel"
      :editor-path="editorPath"
      :sg-label-fn="sgLabelFn"
      :active-node-count="activeNodeCount"
      @pop="$emit('pop')"
      @goto="$emit('goto', $event)"
    />

    <div class="flex-1" />

    <!-- ====== 中: 撤销/重做 + 加内容(节点/库) + 录制 ====== -->
    <UButton
      size="sm" variant="ghost" color="neutral" icon="i-tabler-arrow-back-up"
      :disabled="!canUndo" :title="t('editor.toolbar.undo')" @click="$emit('undo')"
    />
    <UButton
      size="sm" variant="ghost" color="neutral" icon="i-tabler-arrow-forward-up"
      :disabled="!canRedo" :title="t('editor.toolbar.redo')" @click="$emit('redo')"
    />
    <div class="w-px h-5 bg-default mx-1" />
    <UButton
      size="sm" variant="ghost" color="neutral" icon="i-tabler-grid-dots"
      :title="t('editor.toolbar.node_explorer')" @click="$emit('open-node-explorer')"
    />
    <UButton
      size="sm" variant="ghost" color="neutral" icon="i-tabler-books"
      :title="t('editor.toolbar.library_explorer')" @click="$emit('open-library-explorer')"
    />
    <div class="w-px h-5 bg-default mx-1" />
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

    <div class="flex-1" />

    <!-- ====== 右: 运行状态 + 主操作 + ⋯/设置 + 折叠 inspector ====== -->
    <div
      v-if="execStoreRunning"
      class="inline-flex items-center gap-2 rounded-md bg-primary/15 border border-primary/40 px-2 py-0.5 text-[11px] text-primary"
    >
      <span class="size-1.5 rounded-full bg-primary animate-pulse" />
      <span>{{ t('editor.toolbar.running') }}</span>
      <span v-if="runningNodeKind" class="text-primary/80">· {{ runningNodeLabel }}</span>
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

    <!-- ⋯ 更多: 低频 (自动布局 / 吸附 / 连线样式 / 新窗口 / 重载 / 帮助) -->
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
import ContainerEditorBreadcrumb from '@/components/containers/ContainerEditorBreadcrumb.vue'

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
  execStoreRunning: boolean
  runningNodeKind: string | undefined
  runningNodeLabel: string
  dirty: boolean
  canUndo?: boolean
  canRedo?: boolean
  snapEnabled?: boolean
  edgeStyle?: 'default' | 'smoothstep' | 'step'
  isStandalone?: boolean
  // 面包屑身份 (单行布局：面包屑内嵌工具栏左侧, 不再独立成行)
  rootLabel?: string
  editorPath: readonly string[]
  sgLabelFn: (id: string) => string
  activeNodeCount: number | null
}>()

const emit = defineEmits<{
  'update:paletteCollapsed': [v: boolean]
  'update:inspectorCollapsed': [v: boolean]
  // 'record' 带 mode 参数: 'precise' | 'simple'
  'record': [mode: 'precise' | 'simple']
  'stop-record': []
  'cancel-countdown': []
  'try-run': []
  'stop-run': []
  'save': []
  'reload': []
  'validate': []
  'auto-layout': [direction: 'LR' | 'TB']
  // 工具栏按钮 emits (实际 modal 在父 ContainerEditorView 里挂):
  'open-node-explorer': []
  'open-library-explorer': []
  'open-settings': []
  'open-help': []
  'undo': []
  'redo': []
  'toggle-snap': []
  'set-edge-style': [v: 'default' | 'smoothstep' | 'step']
  'open-new-window': []
  'back-to-list': []
  // 面包屑层级导航
  'pop': []
  'goto': [idx: number]
}>()

// ⋯ 更多菜单: 低频操作分组 — 布局 / 吸附 / 连线样式 / 系统 / 帮助.
const moreMenuItems = computed(() => {
  const cur = props.edgeStyle ?? 'default'
  const edgeItem = (v: 'default' | 'smoothstep' | 'step', labelKey: string, icon: string) => ({
    label: t(labelKey),
    icon,
    type: 'checkbox' as const,
    checked: cur === v,
    onUpdateChecked: (c: boolean) => { if (c) emit('set-edge-style', v) },
  })
  const system: { label: string; icon: string; onSelect: () => void }[] = []
  if (!props.isStandalone) {
    system.push({ label: t('editor.toolbar.open_new_window'), icon: 'i-tabler-external-link', onSelect: () => emit('open-new-window') })
  }
  system.push({ label: t('editor.toolbar.reload'), icon: 'i-tabler-refresh', onSelect: () => emit('reload') })

  return [
    [
      {
        label: t('editor.toolbar.auto_layout'),
        icon: 'i-tabler-layout-grid',
        children: [
          { label: t('editor.toolbar.layout_lr'), icon: 'i-tabler-layout-rows', onSelect: () => emit('auto-layout', 'LR') },
          { label: t('editor.toolbar.layout_tb'), icon: 'i-tabler-layout-columns', onSelect: () => emit('auto-layout', 'TB') },
        ],
      },
      {
        label: t('editor.toolbar.edge_style'),
        icon: 'i-tabler-vector-spline',
        children: [
          edgeItem('default', 'editor.toolbar.edge_style_bezier', 'i-tabler-vector-spline'),
          edgeItem('smoothstep', 'editor.toolbar.edge_style_smoothstep', 'i-tabler-vector-bezier-2'),
          edgeItem('step', 'editor.toolbar.edge_style_step', 'i-tabler-line'),
        ],
      },
      {
        label: t('editor.toolbar.snap'),
        icon: props.snapEnabled ? 'i-tabler-magnet' : 'i-tabler-magnet-off',
        type: 'checkbox' as const,
        checked: !!props.snapEnabled,
        onUpdateChecked: () => emit('toggle-snap'),
      },
    ],
    system,
    [
      { label: t('editor.help.title'), icon: 'i-tabler-help-circle', onSelect: () => emit('open-help') },
    ],
  ]
})
</script>
