<!-- Edge right-click menu. -->
<template>
  <template v-if="open">
    <div class="fixed inset-0 z-40" @click="close" @contextmenu.prevent="close" />
    <div
      class="ctx-menu fixed z-50 bg-default border border-default rounded-lg shadow-2xl py-2 min-w-[260px]"
      :style="positionStyle"
      @click.stop
      @contextmenu.prevent
    >
      <div class="ctx-header px-3 py-1.5 mb-1">
        <div class="flex items-center gap-2">
          <UIcon name="i-tabler-arrow-right" class="size-4 text-primary shrink-0" />
          <div class="flex-1 min-w-0">
            <div class="text-[11px] text-dimmed">连接边</div>
            <div class="text-[11px] font-mono text-default truncate">
              <span class="text-primary">{{ edge.from }}</span>
              <span class="text-dimmed mx-1">→</span>
              <span class="text-primary">{{ edge.to }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="ctx-divider" />

      <button
        v-for="item in items"
        :key="item.key"
        type="button"
        class="ctx-item"
        :class="item.colorClass"
        @click="onClick(item.key)"
      >
        <UIcon :name="item.icon" class="size-3.5 shrink-0" />
        <span class="flex-1 text-left">{{ item.label }}</span>
        <span v-if="item.shortcut" class="ctx-shortcut">{{ item.shortcut }}</span>
      </button>
    </div>
  </template>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { GraphEdge } from '@/lib/backend'

export type EdgeMenuAction = 'delete'

const props = defineProps<{
  open: boolean
  position: { x: number; y: number }
  edge: GraphEdge
}>()
const emit = defineEmits<{
  'update:open': [v: boolean]
  action: [a: EdgeMenuAction]
}>()

const positionStyle = computed(() => ({ left: `${props.position.x}px`, top: `${props.position.y}px` }))

const items = computed(() => [
  {
    key: 'delete' as const,
    label: '删除此边',
    icon: 'i-tabler-trash',
    shortcut: 'Del',
    colorClass: 'text-rose-400',
  },
])

function onClick(key: EdgeMenuAction) {
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
