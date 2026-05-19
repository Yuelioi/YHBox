<!-- frontend/src/components/containers/menus/PinContextMenu.vue -->
<!-- Editor v2 C — pin (handle) right-click menu.
     Spec: editor-v2-quick-actions-design.md §2.5. -->
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
        <UIcon name="i-tabler-circle" class="size-3 inline mr-1" />
        pin: <span class="text-primary">{{ pin.nodeID }}.{{ pin.pinName }}</span>
        <span v-if="pin.pinType" class="text-[9px] ml-1">({{ pin.side }}, {{ pin.pinType }})</span>
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
      </button>
    </div>
  </template>
</template>

<script setup lang="ts">
import { computed } from 'vue'

export interface PinInfo {
  nodeID: string
  pinName: string
  side: 'input' | 'output'
  pinType?: string  // 'number' | 'bool' | 'string' | 'point' | 'any' | undefined (exec)
  edgeCount: number  // how many edges currently attached
}

export type PinMenuAction =
  | 'break-all-connections'
  | 'promote-to-var'
  | 'reset-to-literal'
  | 'show-schema'

const props = defineProps<{
  open: boolean
  position: { x: number; y: number }
  pin: PinInfo
}>()
const emit = defineEmits<{
  'update:open': [v: boolean]
  action: [a: PinMenuAction]
}>()

const positionStyle = computed(() => ({ left: `${props.position.x}px`, top: `${props.position.y}px` }))

const items = computed(() => {
  const arr: Array<{ key: PinMenuAction; label: string; color?: string }> = []
  if (props.pin.edgeCount > 0) {
    arr.push({ key: 'break-all-connections', label: `断开所有连接 (${props.pin.edgeCount} 条)`, color: 'text-error' })
  }
  if (props.pin.side === 'input' && props.pin.pinType) {
    arr.push({ key: 'promote-to-var', label: '⚙ 提升为变量 (Promote)', color: 'text-amber-400' })
    arr.push({ key: 'reset-to-literal', label: '设为 literal (断开 + 默认值)' })
  }
  arr.push({ key: 'show-schema', label: '🔗 显示 pin schema 详情', color: 'text-primary' })
  return arr
})

function onClick(key: PinMenuAction) {
  emit('action', key)
  close()
}

function close() {
  emit('update:open', false)
}
</script>
