<!-- Single-node right-click menu. -->
<template>
  <template v-if="open">
    <div
      class="fixed inset-0 z-40"
      @click="close"
      @contextmenu.prevent="close"
    />
    <div
      class="ctx-menu fixed z-50 bg-default border border-default rounded-lg shadow-2xl py-2 min-w-[240px]"
      :style="positionStyle"
      @click.stop
      @contextmenu.prevent
    >
      <!-- Header -->
      <div class="ctx-header px-3 py-1.5 mb-1">
        <div class="flex items-center gap-2">
          <UIcon :name="iconForKind" class="size-4 text-primary shrink-0" />
          <div class="flex-1 min-w-0">
            <div class="text-[12px] font-semibold text-default truncate">
              {{ node.label || kindLabel }}
            </div>
            <div class="text-[10px] text-dimmed font-mono truncate">
              <span v-if="node.label">{{ kindLabel }} · </span>{{ node.id }}
            </div>
          </div>
        </div>
      </div>

      <div class="ctx-divider" />

      <!-- 通用操作 -->
      <button
        v-for="item in commonItems"
        :key="item.key"
        type="button"
        class="ctx-item"
        @click="onClick(item.key)"
      >
        <UIcon :name="item.icon" class="size-3.5 shrink-0 text-dimmed" />
        <span class="flex-1 text-left">{{ item.label }}</span>
        <span v-if="item.shortcut" class="ctx-shortcut">{{ item.shortcut }}</span>
      </button>

      <div class="ctx-divider" />

      <!-- 特殊操作 (kind-specific + state-aware) -->
      <button
        v-for="item in specialItems"
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
import type { GraphNode } from '@/lib/backend'
import { getSpec } from '@/components/containers/nodeRegistry/registry'
import { pinsFor } from '@/components/containers/pinSpec'

const { t } = useI18n()

export type NodeMenuAction =
  | 'copy' | 'cut' | 'paste' | 'duplicate' | 'delete' | 'hard-delete'
  | 'toggle-disable' | 'save-as-snippet'
  | 'find-references' | 'promote-to-var' | 'jump-to-subgraph'
  | 'to-script' | 'debug-from-here'

const props = defineProps<{
  open: boolean
  position: { x: number; y: number }
  node: GraphNode
}>()

const emit = defineEmits<{
  'update:open': [v: boolean]
  action: [a: NodeMenuAction]
}>()

const positionStyle = computed(() => ({
  left: `${props.position.x}px`,
  top: `${props.position.y}px`,
}))

// spec.labelZh 是 i18n key, 经 t() 渲染; spec 缺失时 fallback 到 kind 字面.
const kindLabel = computed(() => {
  const s = getSpec(props.node.kind)
  return s?.labelZh ? t(s.labelZh) : props.node.kind
})

const iconForKind = computed(() => getSpec(props.node.kind)?.visual?.icon ?? 'i-tabler-box')

const isVarRef = computed(() =>
  ['GetVar', 'SetVar', 'IncVar'].includes(props.node.kind),
)

const isSubgraph = computed(() =>
  ['Subgraph', 'CollapsedNode'].includes(props.node.kind),
)

// 有底层全局定义、删节点不会连带删的: 具名 Subgraph (子图池) / PlayClip (clip 资产).
// 这俩才提供「彻底删除」连定义删. CollapsedNode 的后备子图是匿名、删节点后自动 GC, 不用.
const hasUnderlyingDef = computed(() =>
  ['Subgraph', 'PlayClip'].includes(props.node.kind),
)

const hasLiteralPin = computed(() => {
  const lit = (props.node.config as Record<string, unknown> | undefined)?.literal as
    | Record<string, unknown>
    | undefined
  return lit && Object.keys(lit).length > 0
})

const isDisabled = computed(() => (props.node as GraphNode & { disabled?: boolean }).disabled === true)
const canDebugFromHere = computed(() => pinsFor(
  props.node.kind,
  props.node.config as Record<string, unknown> | undefined,
).execIn.length > 0)

const commonItems = computed(() => [
  { key: 'copy' as const, label: t('editor.menu.node.copy'), icon: 'i-tabler-copy', shortcut: 'Ctrl+C' },
  { key: 'cut' as const, label: t('editor.menu.node.cut'), icon: 'i-tabler-cut', shortcut: 'Ctrl+X' },
  { key: 'paste' as const, label: t('editor.menu.node.paste'), icon: 'i-tabler-clipboard', shortcut: 'Ctrl+V' },
  { key: 'duplicate' as const, label: t('editor.menu.node.duplicate'), icon: 'i-tabler-stack-2', shortcut: 'Ctrl+D' },
  { key: 'delete' as const, label: t('editor.menu.node.delete'), icon: 'i-tabler-trash', shortcut: 'Del', colorClass: 'text-error' },
])

const specialItems = computed(() => {
  const items: Array<{ key: NodeMenuAction; label: string; icon: string; colorClass?: string }> = []

  if (canDebugFromHere.value) {
    items.push({
      key: 'debug-from-here',
      label: t('editor.menu.node.debug_from_here'),
      icon: 'i-tabler-bug',
      colorClass: 'text-primary',
    })
  }

  items.push(
    {
      key: 'toggle-disable',
      label: isDisabled.value ? t('editor.menu.node.enable') : t('editor.menu.node.disable'),
      icon: isDisabled.value ? 'i-tabler-player-play' : 'i-tabler-ban',
      colorClass: 'text-warning',
    },
    {
      key: 'save-as-snippet',
      label: t('editor.menu.node.save_as_snippet'),
      icon: 'i-tabler-bookmark-plus',
      colorClass: 'text-yellow-300',
    },
  )

  if (isVarRef.value) {
    const varName = (props.node.config as Record<string, unknown> | undefined)?.varName as
      | string
      | undefined ?? '?'
    items.push({
      key: 'find-references',
      label: t('editor.menu.node.find_var_refs', { name: varName }),
      icon: 'i-tabler-link',
      colorClass: 'text-cyan-300',
    })
  }

  if (hasLiteralPin.value) {
    items.push({
      key: 'promote-to-var',
      label: t('editor.menu.node.promote_to_var'),
      icon: 'i-tabler-variable',
      colorClass: 'text-violet-300',
    })
  }

  if (isSubgraph.value) {
    items.push({
      key: 'jump-to-subgraph',
      label: t('editor.menu.node.enter_subgraph'),
      icon: 'i-tabler-corner-down-right',
      colorClass: 'text-sky-300',
    })
    items.push({
      key: 'to-script',
      label: t('editor.menu.node.to_script'),
      icon: 'i-tabler-code',
      colorClass: 'text-sky-300',
    })
  }

  if (hasUnderlyingDef.value) {
    items.push({
      key: 'hard-delete',
      label: t('editor.menu.node.hard_delete'),
      icon: 'i-tabler-trash-x',
      colorClass: 'text-error',
    })
  }

  return items
})

function onClick(key: NodeMenuAction) {
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
  /* glassmorphism: 跟节点同款 — 内描边高光 + drop shadow */
  box-shadow:
    0 16px 48px -12px rgba(0, 0, 0, 0.7),
    0 4px 10px -2px rgba(0, 0, 0, 0.4),
    inset 0 1px 0 0 rgba(255, 255, 255, 0.06);
  backdrop-filter: blur(6px);
}
.ctx-header {
  /* 跟节点 header 同款渐变 */
  background-image: linear-gradient(
    135deg,
    rgba(255, 255, 255, 0.06) 0%,
    transparent 60%
  );
}
.ctx-divider {
  height: 1px;
  margin: 4px 8px;
  background: linear-gradient(
    90deg,
    transparent,
    rgba(255, 255, 255, 0.1),
    transparent
  );
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
