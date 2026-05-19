<!-- frontend/src/components/containers/menus/MultiNodeContextMenu.vue -->
<!-- Editor v2 C — multi-selection right-click menu.
     Spec: editor-v2-quick-actions-design.md §2.3. -->
<template>
  <template v-if="open">
    <div
      class="fixed inset-0 z-40"
      @click="close"
      @contextmenu.prevent="close"
    />
    <div
      class="fixed z-50 bg-default border border-default rounded shadow-xl py-1 min-w-[220px]"
      :style="positionStyle"
      @click.stop
      @contextmenu.prevent
    >
      <!-- Header -->
      <div class="px-3 py-1 text-[10px] text-dimmed border-b border-default">
        已选 <strong class="text-primary">{{ count }}</strong> 个节点
      </div>

      <button
        v-for="item in items"
        :key="item.key"
        type="button"
        class="w-full text-left px-3 py-1 text-[11px] hover:bg-elevated/60 flex items-center justify-between gap-3"
        :class="item.color"
        @click="onClick(item.key)"
      >
        <span>{{ item.label }}</span>
        <span v-if="item.shortcut" class="text-[10px] text-dimmed">{{ item.shortcut }}</span>
      </button>
    </div>
  </template>
</template>

<script setup lang="ts">
import { computed } from 'vue'

export type MultiMenuAction =
  | 'copy' | 'cut' | 'paste' | 'duplicate' | 'delete'
  | 'toggle-disable-all'
  | 'fold' | 'comment-box'
  | 'align-left' | 'align-right' | 'align-top' | 'align-bottom'
  | 'align-center-h' | 'align-center-v'
  | 'distribute-h' | 'distribute-v'
  | 'auto-layout-lr' | 'auto-layout-tb'

const props = defineProps<{
  open: boolean
  position: { x: number; y: number }
  count: number
}>()

const emit = defineEmits<{
  'update:open': [v: boolean]
  action: [a: MultiMenuAction]
}>()

const positionStyle = computed(() => ({
  left: `${props.position.x}px`,
  top: `${props.position.y}px`,
}))

const items = computed(() => {
  const arr: Array<{ key: MultiMenuAction; label: string; shortcut?: string; color?: string }> = [
    { key: 'copy', label: '复制', shortcut: 'Ctrl+C' },
    { key: 'duplicate', label: '复刻', shortcut: 'Ctrl+D' },
    { key: 'delete', label: '删除', shortcut: 'Del' },
    {
      key: 'toggle-disable-all',
      label: '⊘ 禁用所有 (运行时跳过)',
      color: 'text-warning',
    },
    { key: 'fold', label: '📦 折叠为子图' },
  ]

  if (props.count >= 2) {
    arr.push(
      { key: 'align-left', label: '⫶⫶ 对齐 - 左' },
      { key: 'align-right', label: '⫶⫶ 对齐 - 右' },
      { key: 'align-top', label: '⫶⫶ 对齐 - 顶' },
      { key: 'align-bottom', label: '⫶⫶ 对齐 - 底' },
      { key: 'align-center-h', label: '⫶⫶ 水平居中' },
      { key: 'align-center-v', label: '⫶⫶ 垂直居中' },
    )
  }

  if (props.count >= 3) {
    arr.push(
      { key: 'distribute-h', label: '⊞ 水平等距分布' },
      { key: 'distribute-v', label: '⊞ 垂直等距分布' },
    )
  }

  arr.push(
    { key: 'auto-layout-lr', label: '⊟ 自动布局 (横向)' },
    { key: 'auto-layout-tb', label: '⊟ 自动布局 (纵向)' },
  )

  return arr
})

function onClick(key: MultiMenuAction) {
  emit('action', key)
  close()
}

function close() {
  emit('update:open', false)
}
</script>
