<!-- frontend/src/components/containers/menus/EdgeContextMenu.vue -->
<!-- Editor v2 C — edge right-click menu.
     Spec: editor-v2-quick-actions-design.md §2.4. -->
<template>
  <template v-if="open">
    <div class="fixed inset-0 z-40" @click="close" @contextmenu.prevent="close" />
    <div
      class="fixed z-50 bg-default border border-default rounded shadow-xl py-1 min-w-[220px]"
      :style="positionStyle"
      @click.stop
      @contextmenu.prevent
    >
      <div class="px-3 py-1 text-[10px] text-dimmed border-b border-default">
        <UIcon name="i-tabler-arrow-right" class="size-3 inline mr-1" />
        edge: <span class="text-primary">{{ edge.from }}</span> → <span class="text-primary">{{ edge.to }}</span>
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
import type { GraphEdge } from '@/lib/backend'

export type EdgeMenuAction = 'delete' | 'insert-node-on-edge'

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
  { key: 'delete' as const, label: '删除此边', shortcut: 'Del', color: 'text-error' },
  { key: 'insert-node-on-edge' as const, label: '⊕ 在边上插入节点... (弹 InlineContextMenu)', color: 'text-primary', shortcut: undefined },
])

function onClick(key: EdgeMenuAction) {
  emit('action', key)
  close()
}

function close() {
  emit('update:open', false)
}
</script>
