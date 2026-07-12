<!-- Pin (handle) right-click menu. -->
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
          <UIcon
            :name="pin.pinType ? 'i-tabler-circle-dot' : 'i-tabler-triangle'"
            class="size-4 shrink-0"
            :style="{ color: pinColor }"
          />
          <div class="flex-1 min-w-0">
            <div class="text-[11px] text-dimmed">
              {{
                pin.side === 'input'
                  ? t('editor.menu.pin.title_in')
                  : t('editor.menu.pin.title_out')
              }}
              <span v-if="pin.pinType" class="text-toned">· {{ pin.pinType }}</span>
            </div>
            <div class="text-[11px] font-mono text-default truncate">
              <span class="text-dimmed">{{ pin.nodeID }}</span
              ><span class="text-dimmed">.</span><span class="text-primary">{{ pin.pinName }}</span>
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
      </button>
    </div>
  </template>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { TYPE_COLOR } from '@/components/containers/nodeRegistry/index'
import type { PinType } from '@/components/containers/nodeRegistry/index'

const { t } = useI18n()

export interface PinInfo {
  nodeID: string
  pinName: string
  side: 'input' | 'output'
  pinType?: string // 'number' | 'bool' | 'string' | 'point' | 'any' | undefined (exec)
  edgeCount: number
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

const positionStyle = computed(() => ({
  left: `${props.position.x}px`,
  top: `${props.position.y}px`,
}))

const pinColor = computed(() => {
  if (!props.pin.pinType) return '#e5e7eb' // exec 灰白
  return TYPE_COLOR[props.pin.pinType as PinType] ?? '#9ca3af'
})

const items = computed(() => {
  const arr: Array<{ key: PinMenuAction; label: string; icon: string; colorClass?: string }> = []
  if (props.pin.edgeCount > 0) {
    arr.push({
      key: 'break-all-connections',
      label: t('editor.menu.pin.disconnect_all', { n: props.pin.edgeCount }),
      icon: 'i-tabler-link-off',
      colorClass: 'text-error',
    })
  }
  if (props.pin.side === 'input' && props.pin.pinType) {
    arr.push({
      key: 'promote-to-var',
      label: t('editor.menu.pin.promote_to_var'),
      icon: 'i-tabler-variable',
      colorClass: 'text-violet-300',
    })
    arr.push({
      key: 'reset-to-literal',
      label: t('editor.menu.pin.set_literal'),
      icon: 'i-tabler-arrow-back-up',
    })
  }
  arr.push({
    key: 'show-schema',
    label: t('editor.menu.pin.show_schema'),
    icon: 'i-tabler-info-circle',
    colorClass: 'text-sky-300',
  })
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

<style scoped>
.ctx-menu {
  font-family:
    system-ui,
    -apple-system,
    'Segoe UI Variable Text',
    'PingFang SC',
    'Microsoft YaHei',
    sans-serif;
  box-shadow:
    0 16px 48px -12px rgba(0, 0, 0, 0.7),
    0 4px 10px -2px rgba(0, 0, 0, 0.4),
    inset 0 1px 0 0 rgba(255, 255, 255, 0.06);
  backdrop-filter: blur(6px);
}
.ctx-header {
  background-image: linear-gradient(135deg, rgba(255, 255, 255, 0.06) 0%, transparent 60%);
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
  transition:
    background 120ms ease,
    color 120ms ease;
}
.ctx-item:hover {
  background: rgba(255, 255, 255, 0.06);
}
.ctx-item:active {
  background: rgba(255, 255, 255, 0.1);
}
</style>
