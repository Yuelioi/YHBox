<!-- Multi-selection right-click menu. -->
<template>
  <template v-if="open">
    <div
      class="fixed inset-0 z-40"
      @click="close"
      @contextmenu.prevent="close"
    />
    <div
      class="ctx-menu fixed z-50 bg-default border border-default rounded-lg shadow-2xl py-2 min-w-[260px]"
      :style="positionStyle"
      @click.stop
      @contextmenu.prevent
    >
      <!-- Header -->
      <div class="ctx-header px-3 py-1.5 mb-1">
        <div class="flex items-center gap-2">
          <UIcon name="i-tabler-checkbox" class="size-4 text-primary shrink-0" />
          <div class="text-[12px] font-semibold text-default">
            已选 <span class="text-primary">{{ count }}</span> 个节点
          </div>
        </div>
      </div>

      <div class="ctx-divider" />

      <button
        v-for="item in items"
        :key="item.key + item.label"
        type="button"
        class="ctx-item"
        :class="item.colorClass"
        @click="onClick(item.key)"
      >
        <UIcon :name="item.icon" class="size-3.5 shrink-0" :class="item.iconColor ?? 'text-dimmed'" />
        <span class="flex-1 text-left">{{ item.label }}</span>
        <span v-if="item.shortcut" class="ctx-shortcut">{{ item.shortcut }}</span>
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

interface Item {
  key: MultiMenuAction
  label: string
  icon: string
  shortcut?: string
  colorClass?: string
  iconColor?: string
}

const items = computed(() => {
  const arr: Item[] = [
    { key: 'copy', label: '复制', icon: 'i-tabler-copy', shortcut: 'Ctrl+C' },
    { key: 'duplicate', label: '复刻', icon: 'i-tabler-stack-2', shortcut: 'Ctrl+D' },
    { key: 'delete', label: '删除', icon: 'i-tabler-trash', shortcut: 'Del', colorClass: 'text-rose-400' },
    { key: 'toggle-disable-all', label: '禁用所有 (运行时跳过)', icon: 'i-tabler-ban', colorClass: 'text-amber-400' },
    { key: 'fold', label: '折叠为子图', icon: 'i-tabler-package', colorClass: 'text-violet-300' },
  ]

  if (props.count >= 2) {
    arr.push(
      { key: 'align-left', label: '对齐 - 左', icon: 'i-tabler-align-box-left-middle' },
      { key: 'align-right', label: '对齐 - 右', icon: 'i-tabler-align-box-right-middle' },
      { key: 'align-top', label: '对齐 - 顶', icon: 'i-tabler-align-box-top-center' },
      { key: 'align-bottom', label: '对齐 - 底', icon: 'i-tabler-align-box-bottom-center' },
      { key: 'align-center-h', label: '水平居中', icon: 'i-tabler-align-center' },
      { key: 'align-center-v', label: '垂直居中', icon: 'i-tabler-align-center-vertical' },
    )
  }

  if (props.count >= 3) {
    arr.push(
      { key: 'distribute-h', label: '水平等距分布', icon: 'i-tabler-layout-distribute-horizontal' },
      { key: 'distribute-v', label: '垂直等距分布', icon: 'i-tabler-layout-distribute-vertical' },
    )
  }

  arr.push(
    { key: 'auto-layout-lr', label: '自动布局 (横向)', icon: 'i-tabler-layout-rows', colorClass: 'text-sky-300' },
    { key: 'auto-layout-tb', label: '自动布局 (纵向)', icon: 'i-tabler-layout-columns', colorClass: 'text-sky-300' },
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

<style scoped>
.ctx-menu {
  font-family:
    system-ui, -apple-system, 'Segoe UI Variable Text', 'PingFang SC', 'Microsoft YaHei',
    sans-serif;
  box-shadow:
    0 16px 48px -12px rgba(0, 0, 0, 0.7),
    0 4px 10px -2px rgba(0, 0, 0, 0.4),
    inset 0 1px 0 0 rgba(255, 255, 255, 0.06);
  backdrop-filter: blur(6px);
}
.ctx-header {
  background-image: linear-gradient(
    135deg,
    rgba(255, 255, 255, 0.06) 0%,
    transparent 60%
  );
}
.ctx-divider {
  height: 1px;
  margin: 4px 8px;
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.1), transparent);
}
.ctx-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 6px 12px;
  font-size: 12px;
  color: var(--ui-text-default);
  background: transparent;
  border: none;
  cursor: pointer;
  transition: background 120ms ease, color 120ms ease;
}
.ctx-item:hover {
  background: rgba(255, 255, 255, 0.06);
}
.ctx-item:active {
  background: rgba(255, 255, 255, 0.1);
}
.ctx-shortcut {
  font-size: 10.5px;
  color: var(--ui-text-dimmed);
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  letter-spacing: 0.3px;
  padding: 1px 5px;
  border-radius: 3px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.08);
}
</style>
