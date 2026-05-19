<template>
  <div class="shrink-0 h-11 px-3 border-b border-default flex items-center gap-1 bg-default/60">
    <!-- ====== 左组: sidebar + drag-source openers + undo/redo ====== -->
    <UButton
      size="xs" variant="ghost" color="neutral"
      :icon="paletteCollapsed ? 'i-tabler-layout-sidebar-left-expand' : 'i-tabler-layout-sidebar-left-collapse'"
      :title="paletteCollapsed ? '展开左侧栏' : '折叠左侧栏'"
      @click="$emit('update:paletteCollapsed', !paletteCollapsed)"
    />
    <UButton
      size="sm" variant="ghost" color="neutral"
      icon="i-tabler-grid-dots"
      title="节点 Explorer (Tab)"
      @click="$emit('open-node-explorer')"
    />
    <UButton
      size="sm" variant="ghost" color="neutral"
      icon="i-tabler-books"
      title="子图库 Explorer"
      @click="$emit('open-library-explorer')"
    />
    <UButton
      size="sm" variant="ghost" color="neutral"
      icon="i-tabler-arrow-back-up"
      :disabled="!canUndo"
      title="撤销 Ctrl+Z"
      @click="$emit('undo')"
    />
    <UButton
      size="sm" variant="ghost" color="neutral"
      icon="i-tabler-arrow-forward-up"
      :disabled="!canRedo"
      title="重做 Ctrl+Y"
      @click="$emit('redo')"
    />

    <div class="w-px h-5 bg-default mx-1" />

    <!-- ====== 中组: 录制 + 折叠子图 + 自动布局 ====== -->
    <template v-if="isRecording">
      <UButton size="sm" color="error" variant="solid" icon="i-tabler-square"
               title="点这里 或在游戏前台按 F12 停止"
               @click="$emit('stop-record')">停止录制</UButton>
    </template>
    <template v-else-if="countdownSec > 0">
      <UButton size="sm" color="warning" variant="solid" icon="i-tabler-x"
               title="点这里取消录制倒计时"
               @click="$emit('cancel-countdown')">取消 ({{ countdownSec }})</UButton>
    </template>
    <template v-else>
      <UButton size="sm" color="primary" variant="soft" icon="i-tabler-circle-dot"
               :disabled="!gameDetected"
               :title="!gameDetected ? '游戏窗口未检测到 — 启动游戏后侧栏底部点重新检测' : '精准录制 — 完整事件流 (含鼠标移动/raw delta)'"
               @click="$emit('record', 'precise')">精准录制</UButton>
      <UButton size="sm" color="neutral" variant="subtle" icon="i-tabler-bolt"
               :disabled="!gameDetected"
               :title="!gameDetected ? '游戏窗口未检测到' : '简易录制 — 过滤鼠标 hover / 仅保留 click/key 事件'"
               @click="$emit('record', 'simple')">简易录制</UButton>
    </template>
    <UButton size="sm" variant="soft" color="neutral" icon="i-tabler-package-import"
             :disabled="selectedCount === 0"
             title="折叠选中节点为新子图"
             @click="$emit('fold')">折叠为子图</UButton>
    <UDropdownMenu :items="layoutMenuItems">
      <UButton size="sm" variant="ghost" color="neutral" icon="i-tabler-layout-grid"
               title="自动布局 / 对齐" />
    </UDropdownMenu>
    <UButton
      size="sm"
      variant="ghost"
      :color="snapEnabled ? 'primary' : 'neutral'"
      :icon="snapEnabled ? 'i-tabler-magnet' : 'i-tabler-magnet-off'"
      :title="snapEnabled ? '吸附对齐开启 (拖动按 Alt 临时关) — 点击关闭' : '吸附对齐关闭 — 点击开启'"
      @click="$emit('toggle-snap')"
    />

    <div class="flex-1" />

    <!-- ====== 右组: 运行状态 + 主操作 + 折叠 inspector ====== -->
    <div
      v-if="execStoreRunning"
      class="inline-flex items-center gap-2 rounded-md bg-emerald-500/15 border border-emerald-500/40 px-2 py-0.5 text-[11px] text-emerald-300"
    >
      <span class="size-1.5 rounded-full bg-emerald-400 animate-pulse" />
      <span>跑中</span>
      <span v-if="runningNodeKind" class="text-emerald-200/80">· {{ runningNodeLabel }}</span>
    </div>
    <UButton v-if="execStoreRunning" size="sm" color="error" variant="solid" icon="i-tabler-square"
             title="停止当前运行 + 清队列 (同 Ctrl+Shift+F9)"
             @click="$emit('stop-run')">停止</UButton>

    <UButton size="sm" variant="soft" color="neutral" icon="i-tabler-checks"
             :disabled="dirty"
             :title="dirty ? '请先保存再检查（校验读持久化版本）' : '校验主图 + 所有子图'"
             @click="$emit('validate')">检查</UButton>
    <UButton size="sm" variant="soft" color="primary" icon="i-tabler-player-play"
             :disabled="dirty || execStoreRunning"
             :title="dirty ? '请先保存再试运行' : execStoreRunning ? '已有任务在跑，先停' : '入队运行一次'"
             @click="$emit('try-run')">试运行</UButton>
    <UButton size="sm" color="primary" icon="i-tabler-check" :disabled="!dirty"
             @click="$emit('save')">保存</UButton>
    <UButton size="sm" variant="ghost" color="neutral" icon="i-tabler-settings"
             title="容器设置 (Ctrl+,)"
             @click="$emit('open-settings')" />
    <UButton size="xs" variant="ghost" color="neutral"
             :icon="inspectorCollapsed ? 'i-tabler-layout-sidebar-right-expand' : 'i-tabler-layout-sidebar-right-collapse'"
             :title="inspectorCollapsed ? '展开属性面板' : '折叠属性面板'"
             @click="$emit('update:inspectorCollapsed', !inspectorCollapsed)" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useGameStore } from '@/stores/game'

const props = defineProps<{
  paletteCollapsed: boolean
  inspectorCollapsed: boolean
  // recording 三态:  isRecording (后端真的在录) / countdownSec>0 (倒计时中) / 都不是 (空闲)
  isRecording: boolean
  countdownSec: number
  selectedCount: number
  execStoreRunning: boolean
  runningNodeKind: string | undefined
  runningNodeLabel: string
  dirty: boolean
  canUndo?: boolean
  canRedo?: boolean
  snapEnabled?: boolean
}>()

// 录制前置: 没检测到游戏窗口禁止点 (后端 game.HWND check 已防一层, 这里 UX 层先挡)
const gameStore = useGameStore()
const gameDetected = computed(() => !!gameStore.status?.ok)

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
  'validate': []
  'auto-layout': [direction: 'LR' | 'TB']
  'align-selected': [
    mode: 'left' | 'right' | 'top' | 'bottom' | 'center-h' | 'center-v' | 'h-equal' | 'v-equal',
  ]
  // NEW (Editor v2 A Phase 2 — emit-only stubs; B/C wire up):
  'open-node-explorer': []
  'open-library-explorer': []
  'open-settings': []
  'undo': []
  'redo': []
  'toggle-snap': []
}>()

const layoutMenuItems = computed(() => [
  [
    {
      label: '自动布局（横向）',
      icon: 'i-tabler-layout-board-split',
      onSelect: () => emit('auto-layout', 'LR'),
    },
    {
      label: '自动布局（纵向）',
      icon: 'i-tabler-layout-rows',
      onSelect: () => emit('auto-layout', 'TB'),
    },
  ],
  [
    {
      label: '对齐左 (≥2 节点)',
      icon: 'i-tabler-align-box-left-middle',
      disabled: props.selectedCount < 2,
      onSelect: () => emit('align-selected', 'left'),
    },
    {
      label: '对齐右',
      icon: 'i-tabler-align-box-right-middle',
      disabled: props.selectedCount < 2,
      onSelect: () => emit('align-selected', 'right'),
    },
    {
      label: '对齐顶',
      icon: 'i-tabler-align-box-top-center',
      disabled: props.selectedCount < 2,
      onSelect: () => emit('align-selected', 'top'),
    },
    {
      label: '对齐底',
      icon: 'i-tabler-align-box-bottom-center',
      disabled: props.selectedCount < 2,
      onSelect: () => emit('align-selected', 'bottom'),
    },
    {
      label: '水平居中',
      icon: 'i-tabler-align-center-horizontal',
      disabled: props.selectedCount < 2,
      onSelect: () => emit('align-selected', 'center-h'),
    },
    {
      label: '垂直居中',
      icon: 'i-tabler-align-center-vertical',
      disabled: props.selectedCount < 2,
      onSelect: () => emit('align-selected', 'center-v'),
    },
  ],
  [
    {
      label: '水平等距分布 (≥3)',
      icon: 'i-tabler-arrows-horizontal',
      disabled: props.selectedCount < 3,
      onSelect: () => emit('align-selected', 'h-equal'),
    },
    {
      label: '垂直等距分布',
      icon: 'i-tabler-arrows-vertical',
      disabled: props.selectedCount < 3,
      onSelect: () => emit('align-selected', 'v-equal'),
    },
  ],
])
</script>
