<!-- Multi-selection right-click menu. -->
<template>
  <template v-if="open">
    <div class="fixed inset-0 z-40" @click="close" @contextmenu.prevent="close" />
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
          <i18n-t
            keypath="editor.menu.multi.title_selected"
            tag="div"
            class="text-[12px] font-semibold text-default"
          >
            <template #count
              ><span class="text-primary">{{ count }}</span></template
            >
          </i18n-t>
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
        <UIcon
          :name="item.icon"
          class="size-3.5 shrink-0"
          :class="item.iconColor ?? 'text-dimmed'"
        />
        <span class="flex-1 text-left">{{ item.label }}</span>
        <span v-if="item.shortcut" class="ctx-shortcut">{{ item.shortcut }}</span>
      </button>
    </div>
  </template>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

export type MultiMenuAction =
  | 'copy'
  | 'cut'
  | 'paste'
  | 'duplicate'
  | 'delete'
  | 'toggle-disable-all'
  | 'fold'
  | 'align-left'
  | 'align-right'
  | 'align-top'
  | 'align-bottom'
  | 'align-center-h'
  | 'align-center-v'
  | 'distribute-h'
  | 'distribute-v'
  | 'auto-layout-lr'
  | 'auto-layout-tb'

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
    { key: 'copy', label: t('editor.menu.multi.copy'), icon: 'i-tabler-copy', shortcut: 'Ctrl+C' },
    {
      key: 'duplicate',
      label: t('editor.menu.multi.duplicate'),
      icon: 'i-tabler-stack-2',
      shortcut: 'Ctrl+D',
    },
    {
      key: 'delete',
      label: t('editor.menu.multi.delete'),
      icon: 'i-tabler-trash',
      shortcut: 'Del',
      colorClass: 'text-error',
    },
    {
      key: 'toggle-disable-all',
      label: t('editor.menu.multi.disable_all'),
      icon: 'i-tabler-ban',
      colorClass: 'text-warning',
    },
    {
      key: 'fold',
      label: t('editor.menu.multi.fold'),
      icon: 'i-tabler-package',
      colorClass: 'text-violet-300',
    },
  ]

  if (props.count >= 2) {
    arr.push(
      {
        key: 'align-left',
        label: t('editor.menu.multi.align_left'),
        icon: 'i-tabler-align-box-left-middle',
      },
      {
        key: 'align-right',
        label: t('editor.menu.multi.align_right'),
        icon: 'i-tabler-align-box-right-middle',
      },
      {
        key: 'align-top',
        label: t('editor.menu.multi.align_top'),
        icon: 'i-tabler-align-box-top-center',
      },
      {
        key: 'align-bottom',
        label: t('editor.menu.multi.align_bottom'),
        icon: 'i-tabler-align-box-bottom-center',
      },
      {
        key: 'align-center-h',
        label: t('editor.menu.multi.center_h'),
        icon: 'i-tabler-layout-align-middle',
      },
      {
        key: 'align-center-v',
        label: t('editor.menu.multi.center_v'),
        icon: 'i-tabler-layout-align-center',
      },
    )
  }

  if (props.count >= 3) {
    arr.push(
      {
        key: 'distribute-h',
        label: t('editor.menu.multi.dist_h'),
        icon: 'i-tabler-layout-distribute-horizontal',
      },
      {
        key: 'distribute-v',
        label: t('editor.menu.multi.dist_v'),
        icon: 'i-tabler-layout-distribute-vertical',
      },
    )
  }

  arr.push(
    {
      key: 'auto-layout-lr',
      label: t('editor.menu.multi.auto_layout_lr'),
      icon: 'i-tabler-layout-rows',
      colorClass: 'text-sky-300',
    },
    {
      key: 'auto-layout-tb',
      label: t('editor.menu.multi.auto_layout_tb'),
      icon: 'i-tabler-layout-columns',
      colorClass: 'text-sky-300',
    },
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
