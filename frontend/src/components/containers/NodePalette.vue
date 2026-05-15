<template>
  <div class="space-y-2 text-xs">
    <div v-for="group in groups" :key="group.label">
      <div class="font-medium text-toned mb-1">{{ group.label }}</div>
      <div class="space-y-0.5">
        <button
          v-for="n in group.items"
          :key="n.kind"
          type="button"
          class="w-full text-left px-2 py-1 rounded hover:bg-elevated/40 text-default transition-colors cursor-grab active:cursor-grabbing"
          draggable="true"
          @dragstart="onDragStart($event, n.kind)"
          @click="$emit('add', n.kind)"
        >
          <UIcon :name="n.icon" class="size-3.5 mr-1.5 inline text-dimmed" />
          {{ n.label }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineEmits<{ add: [kind: string] }>()

function onDragStart(e: DragEvent, kind: string) {
  if (!e.dataTransfer) return
  e.dataTransfer.effectAllowed = 'copy'
  // 用统一 mime 让 canvas 端识别本组件的 drag
  e.dataTransfer.setData('application/x-yhbox-node', kind)
}

import { KIND_LABEL_ZH } from './pinSpec'

const KINDS_BY_GROUP: {
  label: string
  items: { kind: string; icon: string }[]
}[] = [
  {
    label: '控制流',
    items: [
      { kind: 'Start', icon: 'i-tabler-player-play' },
      { kind: 'Sleep', icon: 'i-tabler-clock' },
      { kind: 'Loop', icon: 'i-tabler-repeat' },
      { kind: 'If', icon: 'i-tabler-git-branch' },
      { kind: 'Parallel', icon: 'i-tabler-columns-3' },
      { kind: 'Race', icon: 'i-tabler-flag' },
      { kind: 'Stop', icon: 'i-tabler-square' },
      { kind: 'Break', icon: 'i-tabler-player-skip-forward' },
      { kind: 'Continue', icon: 'i-tabler-corner-down-left' },
    ],
  },
  {
    label: '变量',
    items: [
      { kind: 'SetVar', icon: 'i-tabler-equal' },
      { kind: 'IncVar', icon: 'i-tabler-circle-plus' },
    ],
  },
  {
    label: '图像',
    items: [
      { kind: 'WaitTemplate', icon: 'i-tabler-eye' },
      { kind: 'CheckTemplate', icon: 'i-tabler-search' },
      { kind: 'ClickTemplate', icon: 'i-tabler-target' },
      { kind: 'DetectColor', icon: 'i-tabler-color-picker' },
    ],
  },
  { label: '动作', items: [{ kind: 'InvokeAction', icon: 'i-tabler-movie' }] },
  {
    label: '输入',
    items: [
      { kind: 'ClickAt', icon: 'i-tabler-click' },
      { kind: 'KeyPress', icon: 'i-tabler-keyboard' },
      { kind: 'MouseMoveRel', icon: 'i-tabler-arrows-move' },
      { kind: 'Scroll', icon: 'i-tabler-mouse' },
    ],
  },
  { label: '事件', items: [{ kind: 'OnEvent', icon: 'i-tabler-radio' }] },
  {
    label: '调试',
    items: [
      { kind: 'Log', icon: 'i-tabler-file-text' },
      { kind: 'Toast', icon: 'i-tabler-bell' },
    ],
  },
]

// 给模板用：补 label 字段（zh 显示）
const groups = KINDS_BY_GROUP.map((g) => ({
  label: g.label,
  items: g.items.map((n) => ({ ...n, label: KIND_LABEL_ZH[n.kind] ?? n.kind })),
}))
</script>
