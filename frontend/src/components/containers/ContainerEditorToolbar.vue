<template>
  <!-- 三区分层：左 [返回 · 面包屑 · 概览 · 撤销/重做] · 中 [录制 · 检测/保存 · 调试/运行] · 右 [自动布局 · ⋯]
       (加内容 节点库/资产 在左侧 rail; 折叠 Inspector 在画布右边缘 toggle) -->
  <div class="shrink-0 h-11 px-3 border-b border-default flex items-center gap-1 bg-default/60">
    <!-- ====== 左: 回列表 + 面包屑身份 + 撤销/重做 ====== -->
    <UButton
      size="xs" variant="ghost" color="neutral"
      icon="i-tabler-arrow-left"
      :title="t('editor.toolbar.back_to_list')"
      @click="$emit('back-to-list')"
    />
    <ContainerEditorBreadcrumb
      class="ml-1 min-w-0"
      :root-label="rootLabel"
      :editor-path="editorPath"
      :sg-label-fn="sgLabelFn"
      :dirty="dirty"
      @goto="$emit('goto', $event)"
    />

    <ContainerOverviewPopover
      class="ml-1"
      :node-count="nodeCount"
      :var-count="varCount"
      :subgraph-count="subgraphCount"
      :hotkey="overviewHotkey"
      @open-help="$emit('open-help')"
    />

    <!-- 撤销/重做: 跟在面包屑后 (标题栏左区)。 -->
    <div class="w-px h-5 bg-default mx-1" />
    <UButton
      size="sm" variant="ghost" color="neutral" icon="i-tabler-arrow-back-up"
      :disabled="!canUndo" :title="t('editor.toolbar.undo')" @click="$emit('undo')"
    />
    <UButton
      size="sm" variant="ghost" color="neutral" icon="i-tabler-arrow-forward-up"
      :disabled="!canRedo" :title="t('editor.toolbar.redo')" @click="$emit('redo')"
    />

    <div class="flex-1" />

    <!-- ====== 中 · 主工作流: 录制 · 检测/保存 · 调试/运行 ====== -->
    <!-- 录制 (三态紧凑单控件): 空闲=下拉选精准·简易 (neutral, 次操作); 倒计时=点取消; 录制中=红停止(目标进 tooltip)。 -->
    <UButton v-if="isRecording" size="sm" color="error" variant="solid" icon="i-tabler-square"
             :title="(recordingTargetName ? t('editor.toolbar.recording_target_tip', { name: recordingTargetName }) + ' · ' : '') + t('editor.toolbar.stop_record_tip', { hk: hotkeys.keyFor('recording.stop', 'F12') })"
             @click="$emit('stop-record')">{{ t('editor.toolbar.stop_record') }}</UButton>
    <UButton v-else-if="countdownSec > 0" size="sm" color="warning" variant="solid" icon="i-tabler-x"
             :title="t('editor.toolbar.cancel_countdown_tip')"
             @click="$emit('cancel-countdown')">{{ t('editor.toolbar.cancel_countdown', { n: countdownSec }) }}</UButton>
    <UDropdownMenu v-else :items="recordMenuItems">
      <UButton size="sm" color="neutral" variant="soft" icon="i-tabler-circle-dot"
               :title="t('editor.toolbar.record_precise') + ' / ' + t('editor.toolbar.record_simple')">
        {{ t('editor.toolbar.record') }}</UButton>
    </UDropdownMenu>

    <div class="w-px h-5 bg-default mx-1" />

    <UButton size="sm" variant="soft" color="neutral" icon="i-tabler-checks"
             :disabled="dirty"
             :title="dirty ? t('editor.toolbar.validate_dirty_tip') : t('editor.toolbar.validate_tip')"
             @click="$emit('validate')">{{ t('editor.toolbar.validate') }}</UButton>

    <!-- 保存 (dirty 黄点 + 成功内联闪「已保存」, 不弹 toast) -->
    <div class="relative">
      <UButton size="sm" :color="saveFlash ? 'success' : 'primary'" :variant="saveFlash ? 'soft' : 'solid'"
               icon="i-tabler-check" :disabled="!dirty && !saveFlash"
               @click="dirty && $emit('save')">
        {{ saveFlash ? t('editor.toolbar.saved') : t('editor.toolbar.save') }}</UButton>
      <span v-if="dirty && !saveFlash"
            class="absolute -top-0.5 -right-0.5 size-2 rounded-full bg-warning ring-2 ring-default" />
    </div>

    <div class="w-px h-5 bg-default mx-1" />

    <!-- 运行 hero / 运行态状态指示 + 停止。hero = primary solid (自动套 btn-primary-raised 绿渐变)，size md 比周围大一档。 -->
    <div
      v-if="execStoreRunning || debugActive"
      class="inline-flex items-center gap-2 rounded-md bg-primary/15 border border-primary/40 px-2 py-0.5 text-[11px] text-primary"
    >
      <span class="size-1.5 rounded-full bg-primary animate-pulse" />
      <span>{{ debugActive ? t('editor.toolbar.debugging') : t('editor.toolbar.running') }}</span>
      <span v-if="activeNodeLabel" class="text-primary/80">· {{ activeNodeLabel }}</span>
    </div>
    <UButton v-if="execStoreRunning" size="sm" color="error" variant="solid" icon="i-tabler-square"
             :title="t('editor.toolbar.stop_run_tip', { hk: hotkeys.keyFor('system.execution-stop', 'Ctrl+Shift+F9') })"
             @click="$emit('stop-run')">{{ t('editor.toolbar.stop_run') }}</UButton>
    <template v-else-if="debugActive">
      <UButton size="sm" color="primary" variant="solid" icon="i-tabler-player-track-next"
               :disabled="!debugCanStep"
               :title="t('editor.toolbar.debug_step_tip')"
               @click="$emit('debug-step')">{{ t('editor.toolbar.debug_step') }}</UButton>
      <UButton size="sm" color="primary" variant="soft" icon="i-tabler-player-play"
               :disabled="!debugCanContinue"
               :title="t('editor.toolbar.debug_continue_tip')"
               @click="$emit('debug-continue')">{{ t('editor.toolbar.debug_continue') }}</UButton>
      <UButton size="sm" color="warning" variant="soft" icon="i-tabler-player-pause"
               :disabled="!debugCanPause"
               :title="t('editor.toolbar.debug_pause_tip')"
               @click="$emit('debug-pause')">{{ t('editor.toolbar.debug_pause') }}</UButton>
      <UButton size="sm" color="error" variant="solid" icon="i-tabler-square"
               :title="t('editor.toolbar.debug_stop_tip')"
               @click="$emit('debug-stop')">{{ t('editor.toolbar.stop_run') }}</UButton>
    </template>
    <template v-else>
      <UButton size="sm" color="neutral" variant="soft" icon="i-tabler-bug"
               class="toolbar-debug-risk"
               :disabled="dirty"
               :title="dirty ? t('editor.toolbar.debug_dirty_tip') : t('editor.toolbar.debug_tip')"
               @click="$emit('debug-start')">{{ t('editor.toolbar.debug') }}</UButton>
      <UButton size="md" color="primary" variant="solid" icon="i-tabler-player-play"
             :disabled="dirty"
             :title="dirty ? t('editor.toolbar.try_run_dirty_tip') : t('editor.toolbar.try_run_tip')"
             @click="$emit('try-run')">{{ t('editor.toolbar.run_hero') }}</UButton>
    </template>

    <div class="flex-1" />

    <!-- ====== 右 · 低频工具: 自动布局 · ⋯ ====== -->
    <!-- 自动布局 (升为直接下拉, 出 ⋯) -->
    <UDropdownMenu :items="layoutMenuItems">
      <UButton size="sm" variant="ghost" color="neutral" icon="i-tabler-layout-grid"
               :title="t('editor.toolbar.auto_layout')" />
    </UDropdownMenu>

    <!-- ⋯ 更多: 重载 / 吸附 / 连线样式 / 设置 / 帮助 -->
    <UDropdownMenu :items="moreMenuItems">
      <UButton size="sm" variant="ghost" color="neutral" icon="i-tabler-dots"
               :title="t('editor.toolbar.more')" />
    </UDropdownMenu>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useHotkeysStore } from '@/stores/hotkeys'
import ContainerEditorBreadcrumb from '@/components/containers/ContainerEditorBreadcrumb.vue'
import ContainerOverviewPopover from '@/components/containers/ContainerOverviewPopover.vue'

const { t } = useI18n()
const hotkeys = useHotkeysStore()

const props = defineProps<{
  // 左区容器概览 popover (节点/变量/子图数 + 热键)。
  nodeCount: number
  varCount: number
  subgraphCount: number
  overviewHotkey: string
  // recording 三态: isRecording (后端真的在录) / countdownSec>0 (倒计时中) / 都不是 (空闲)
  isRecording: boolean
  /** 录制目标容器名 (录制中进停止按钮 tooltip, 空则不拼) */
  recordingTargetName?: string
  countdownSec: number
  execStoreRunning: boolean
  runningNodeKind: string | undefined
  runningNodeLabel: string
  debugActive?: boolean
  debugCanStep?: boolean
  debugCanContinue?: boolean
  debugCanPause?: boolean
  debugRunningNodeKind?: string | undefined
  debugRunningNodeLabel?: string
  dirty: boolean
  /** 保存成功后的短暂窗口 — 保存按钮闪「已保存」(成功反馈内联, 不弹 toast)。 */
  saveFlash?: boolean
  canUndo?: boolean
  canRedo?: boolean
  snapEnabled?: boolean
  edgeStyle?: 'default' | 'smoothstep' | 'step'
  // 面包屑身份 (单行布局：面包屑内嵌工具栏左侧, 不再独立成行)
  rootLabel?: string
  editorPath: readonly string[]
  sgLabelFn: (id: string) => string
}>()

const emit = defineEmits<{
  // 'record' 带 mode 参数: 'precise' | 'simple'
  'record': [mode: 'precise' | 'simple']
  'stop-record': []
  'cancel-countdown': []
  'try-run': []
  'stop-run': []
  'debug-start': []
  'debug-step': []
  'debug-continue': []
  'debug-pause': []
  'debug-stop': []
  'save': []
  'reload': []
  'validate': []
  'auto-layout': [direction: 'LR' | 'TB']
  // 工具栏按钮 emits (实际 modal 在父 ContainerEditorView 里挂):
  'open-settings': []
  'open-js-console': []
  'open-help': []
  'undo': []
  'redo': []
  'toggle-snap': []
  'set-edge-style': [v: 'default' | 'smoothstep' | 'step']
  'back-to-list': []
  // 面包屑层级导航 (goto -1 = 回主图根)
  'goto': [idx: number]
}>()

const activeNodeLabel = computed(() => {
  if (props.debugActive) return props.debugRunningNodeLabel || props.debugRunningNodeKind || ''
  return props.runningNodeLabel || props.runningNodeKind || ''
})

// 录制下拉 (空闲态): 精准 / 简易。
const recordMenuItems = [[
  { label: t('editor.toolbar.record_precise'), icon: 'i-tabler-circle-dot', onSelect: () => emit('record', 'precise') },
  { label: t('editor.toolbar.record_simple'), icon: 'i-tabler-bolt', onSelect: () => emit('record', 'simple') },
]]

// 自动布局下拉 (右区直接按钮, 出 ⋯): 横向 / 纵向。
const layoutMenuItems = [[
  { label: t('editor.toolbar.layout_lr'), icon: 'i-tabler-layout-rows', onSelect: () => emit('auto-layout', 'LR') },
  { label: t('editor.toolbar.layout_tb'), icon: 'i-tabler-layout-columns', onSelect: () => emit('auto-layout', 'TB') },
]]

// ⋯ 更多菜单: 低频收纳 — 连线样式 / 吸附 / 重载 + 设置 / 帮助。(自动布局已出, 设置已进)
const moreMenuItems = computed(() => {
  const cur = props.edgeStyle ?? 'default'
  const edgeItem = (v: 'default' | 'smoothstep' | 'step', labelKey: string, icon: string) => ({
    label: t(labelKey),
    icon,
    type: 'checkbox' as const,
    checked: cur === v,
    onUpdateChecked: (c: boolean) => { if (c) emit('set-edge-style', v) },
  })

  return [
    [
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
    [
      { label: t('editor.toolbar.reload'), icon: 'i-tabler-refresh', onSelect: () => emit('reload') },
      { label: t('editor.palette.cmd.js_console'), icon: 'i-tabler-terminal-2', onSelect: () => emit('open-js-console') },
      { label: t('editor.toolbar.open_settings'), icon: 'i-tabler-settings', onSelect: () => emit('open-settings') },
    ],
    [
      { label: t('editor.help.title'), icon: 'i-tabler-help-circle', onSelect: () => emit('open-help') },
    ],
  ]
})
</script>

<style scoped>
.toolbar-debug-risk :deep(svg),
.toolbar-debug-risk :deep(.iconify) {
  color: var(--ui-warning);
}
</style>
