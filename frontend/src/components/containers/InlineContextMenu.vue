<!-- Inline 浮动 context menu — 画布 add node 入口.
     触发: 右键空白 / 双击空白 / pin 拖出松开.
     Pin-context 时按 pin 类型过滤 compatible nodes. -->
<template>
  <!-- Backdrop to dismiss on outside click (renders below the panel) -->
  <div
    v-if="open"
    class="fixed inset-0 z-40"
    @click="close"
    @contextmenu.prevent="close"
  />

  <div
    v-if="open"
    class="fixed z-50 bg-default border border-primary rounded shadow-xl p-2"
    :style="positionStyle"
    @click.stop
    @contextmenu.prevent
  >
    <!-- Context header -->
    <div class="text-[10px] text-dimmed mb-1.5 px-1">
      <template v-if="pinContext">
        <UIcon name="i-tabler-plus" class="size-3 inline-block" />
        接受 <strong class="text-primary">{{ pinContext.pinType }}</strong> ({{ filtered.length }})
      </template>
      <template v-else>
        添加节点 ({{ filtered.length }})
      </template>
    </div>

    <!-- Search -->
    <UInput
      ref="searchInputRef"
      v-model="query"
      placeholder="type filter..."
      icon="i-tabler-search"
      size="sm"
      class="mb-2"
      @keydown.escape.stop="close"
      @keydown.enter.stop="pickFirst"
    />

    <!-- Tree: per-group collapsible sections -->
    <div class="max-h-64 overflow-y-auto pr-1" style="width: 240px;">
      <template v-if="filtered.length === 0">
        <p class="text-[10px] text-dimmed italic px-1 py-2 text-center">无匹配</p>
      </template>
      <template v-else>
        <div v-for="g in groupedFiltered" :key="g.group">
          <button
            type="button"
            class="w-full flex items-center gap-1 px-1 py-0.5 hover:bg-elevated/30 rounded text-[10px] font-medium text-default"
            @click="toggleGroup(g.group)"
          >
            <UIcon :name="isExpanded(g.group) ? 'i-tabler-chevron-down' : 'i-tabler-chevron-right'" class="size-3" />
            <span>{{ groupLabelZh(g.group) }}</span>
            <span class="text-[9px] opacity-70">({{ g.specs.length }})</span>
          </button>
          <div v-show="isExpanded(g.group)" class="pl-3 space-y-0.5">
            <button
              v-for="spec in g.specs"
              :key="spec.kind"
              type="button"
              class="w-full text-left px-2 py-1 hover:bg-elevated/60 rounded text-[10px] flex items-center gap-1.5"
              :title="spec.description ?? spec.kind"
              @click="pick(spec.kind)"
            >
              <UIcon v-if="spec.visual?.icon" :name="spec.visual.icon" class="size-3 shrink-0" :class="nodeIconColor(spec)" />
              <span class="flex-1 truncate">{{ spec.labelZh ?? spec.kind }}</span>
              <span class="text-[8px] text-dimmed">{{ spec.kind }}</span>
            </button>
          </div>
        </div>
      </template>
    </div>

    <!-- Hint -->
    <div class="text-[9px] text-dimmed mt-1 text-center">Esc 关 · Enter 选首项</div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { allSpecs } from '@/components/containers/nodeRegistry/registry'
import type { NodeKindSpec } from '@/components/containers/nodeRegistry'
import { isCompatibleType, type VarType } from '@/lib/variableRef'
import { nodeIconColor, groupLabelZh } from '@/composables/editor/useNodeGroupColor'

export interface PinContext {
  pinType: VarType
  /** 'output' = user dragged from an output pin → find nodes that accept this type as data-in
   *  'input'  = user dragged from an input pin  → find nodes that produce this type as data-out */
  side: 'input' | 'output'
}

const props = defineProps<{
  open: boolean
  /** Position in viewport (screen) coords */
  position: { x: number; y: number }
  /** Optional pin context for type-aware filtering */
  pinContext?: PinContext
}>()

const emit = defineEmits<{
  'update:open': [v: boolean]
  pick: [kind: string]
}>()

const query = ref('')
const searchInputRef = ref<any>(null)

// Re-focus search and clear query whenever the menu opens
watch(
  () => props.open,
  async (v) => {
    if (v) {
      query.value = ''
      // Reset expand state to all expanded when reopening
      expandedGroups.value = new Set(ALL_GROUPS)
      await nextTick()
      // UInput exposes the native input via .input ref
      searchInputRef.value?.input?.focus?.()
    }
  },
)

const positionStyle = computed(() => ({
  left: `${props.position.x}px`,
  top: `${props.position.y}px`,
  width: 'auto',
}))

/** All specs eligible for insertion (excludes visual-only and palette-excluded kinds) */
const allEligible = computed<NodeKindSpec[]>(() => {
  let specs = allSpecs().filter((s) => !s.isVisualOnly && !s.excludeFromPalette)

  if (!props.pinContext) return specs

  const t = props.pinContext.pinType
  const side = props.pinContext.side

  return specs.filter((s) => {
    if (side === 'output') {
      // User dragged from output pin → need a node that accepts this type as data-in
      if (!s.dataIn || Object.keys(s.dataIn).length === 0) return false
      return Object.values(s.dataIn).some((pt) => isCompatibleType(t, pt as VarType))
    } else {
      // side === 'input' — user dragged from input pin → need a node that produces this type as data-out
      if (!s.dataOut || Object.keys(s.dataOut).length === 0) return false
      return Object.values(s.dataOut).some((pt) => isCompatibleType(pt as VarType, t))
    }
  })
})

const filtered = computed<NodeKindSpec[]>(() => {
  const q = query.value.toLowerCase().trim()
  if (!q) return allEligible.value
  return allEligible.value.filter((s) => {
    const hay = `${s.kind} ${s.labelZh ?? ''} ${s.description ?? ''}`.toLowerCase()
    return hay.includes(q)
  })
})

const groupedFiltered = computed(() => {
  const byGroup = new Map<string, NodeKindSpec[]>()
  for (const s of filtered.value) {
    const g = (s.group ?? 'misc') as string
    if (!byGroup.has(g)) byGroup.set(g, [])
    byGroup.get(g)!.push(s)
  }
  return Array.from(byGroup.entries())
    .map(([group, specs]) => ({
      group,
      specs: specs.sort((a, b) => a.kind.localeCompare(b.kind)),
    }))
    .sort((a, b) => a.group.localeCompare(b.group))
})

// Expand state (transient — no localStorage for popup)
const ALL_GROUPS = ['control', 'variables', 'purefunc', 'detect', 'input', 'system', 'misc']
const expandedGroups = ref<Set<string>>(new Set(ALL_GROUPS))

function toggleGroup(group: string) {
  if (expandedGroups.value.has(group)) expandedGroups.value.delete(group)
  else expandedGroups.value.add(group)
  // Force reactivity since Set mutation doesn't trigger
  expandedGroups.value = new Set(expandedGroups.value)
}

function isExpanded(group: string): boolean {
  if (query.value.trim()) return true  // Auto-expand all when searching
  return expandedGroups.value.has(group)
}

function pick(kind: string) {
  emit('pick', kind)
  close()
}

function pickFirst() {
  const f = filtered.value
  if (f.length > 0) pick(f[0].kind)
}

function close() {
  emit('update:open', false)
}
</script>
