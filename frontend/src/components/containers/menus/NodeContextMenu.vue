<!-- frontend/src/components/containers/menus/NodeContextMenu.vue -->
<!-- Editor v2 C — single-node right-click menu.
     Spec: editor-v2-quick-actions-design.md §2.2. -->
<template>
  <template v-if="open">
    <div
      class="fixed inset-0 z-40"
      @click="close"
      @contextmenu.prevent="close"
    />
    <div
      class="fixed z-50 bg-default border border-default rounded shadow-xl py-1 min-w-[200px]"
      :style="positionStyle"
      @click.stop
      @contextmenu.prevent
    >
      <!-- Header -->
      <div class="px-3 py-1 text-[10px] text-dimmed border-b border-default">
        <UIcon :name="iconForKind" class="size-3 inline mr-1" />
        {{ kindLabel }} <span class="text-primary">({{ node.id }})</span>
      </div>

      <!-- 通用操作 -->
      <button
        v-for="item in commonItems"
        :key="item.key"
        type="button"
        class="w-full text-left px-3 py-1 text-[11px] hover:bg-elevated/60 flex items-center justify-between gap-3"
        @click="onClick(item.key)"
      >
        <span>{{ item.label }}</span>
        <span v-if="item.shortcut" class="text-[10px] text-dimmed">{{ item.shortcut }}</span>
      </button>

      <div class="border-t border-default my-1" />

      <!-- 特殊操作 (kind-specific + state-aware) -->
      <button
        v-for="item in specialItems"
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
import type { GraphNode } from '@/lib/backend'
import { getSpec } from '@/components/containers/nodeRegistry/registry'
import { useDiscoveryStore } from '@/stores/discovery'

export type NodeMenuAction =
  | 'copy' | 'cut' | 'paste' | 'duplicate' | 'delete'
  | 'toggle-disable' | 'star' | 'rename'
  | 'jump-to-var' | 'find-references' | 'promote-to-var' | 'jump-to-subgraph'

const props = defineProps<{
  open: boolean
  position: { x: number; y: number }
  node: GraphNode
}>()

const emit = defineEmits<{
  'update:open': [v: boolean]
  action: [a: NodeMenuAction]
}>()

const discovery = useDiscoveryStore()

const positionStyle = computed(() => ({
  left: `${props.position.x}px`,
  top: `${props.position.y}px`,
}))

const kindLabel = computed(() => {
  const s = getSpec(props.node.kind)
  return s?.labelZh ?? props.node.kind
})

const iconForKind = computed(() => getSpec(props.node.kind)?.visual?.icon ?? 'i-tabler-box')

const isVarRef = computed(() =>
  ['GetVar', 'SetVar', 'IncVar'].includes(props.node.kind),
)

const isSubgraph = computed(() =>
  ['Subgraph', 'CollapsedNode'].includes(props.node.kind),
)

const hasLiteralPin = computed(() => {
  const lit = (props.node.config as Record<string, unknown> | undefined)?.literal as
    | Record<string, unknown>
    | undefined
  return lit && Object.keys(lit).length > 0
})

const isDisabled = computed(() => (props.node as GraphNode & { disabled?: boolean }).disabled === true)

const commonItems = computed(() => [
  { key: 'copy' as const, label: '复制', shortcut: 'Ctrl+C' },
  { key: 'cut' as const, label: '剪切', shortcut: 'Ctrl+X' },
  { key: 'paste' as const, label: '粘贴', shortcut: 'Ctrl+V' },
  { key: 'duplicate' as const, label: '复刻', shortcut: 'Ctrl+D' },
  { key: 'delete' as const, label: '删除', shortcut: 'Del' },
])

const specialItems = computed(() => {
  const items: Array<{ key: NodeMenuAction; label: string; color?: string }> = [
    {
      key: 'toggle-disable',
      label: isDisabled.value ? '✓ 启用此节点' : '⊘ 禁用此节点 (运行时跳过)',
      color: 'text-warning',
    },
    {
      key: 'star',
      label: discovery.favorites.includes(props.node.kind) ? '★ 已收藏' : '☆ 加入收藏',
      color: discovery.favorites.includes(props.node.kind) ? 'text-amber-400' : '',
    },
  ]

  if (isVarRef.value) {
    const varName = (props.node.config as Record<string, unknown> | undefined)?.varName as
      | string
      | undefined ?? '?'
    items.push({
      key: 'jump-to-var',
      label: `↑ 跳到变量 '${varName}' (左 sidebar)`,
      color: 'text-amber-400',
    })
    items.push({
      key: 'find-references',
      label: `🔗 查找所有引用 '${varName}'`,
      color: 'text-amber-400',
    })
  }

  if (hasLiteralPin.value) {
    items.push({
      key: 'promote-to-var',
      label: '⚙ 提取为变量 (Promote)',
      color: 'text-amber-400',
    })
  }

  if (isSubgraph.value) {
    items.push({
      key: 'jump-to-subgraph',
      label: '↪ 进入子图',
      color: 'text-primary',
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
