<!-- 节点库停靠面板 (Houdini 式可折叠树). 入口: rail 节点库图标 / Tab 键 (canvas focused).
     从 NodeExplorerModal 抽 body — 去 modal 外壳, 停靠区自管高度. -->
<template>
  <div class="flex flex-col h-full min-h-0">
    <!-- 标题栏 (停靠区无外层 header, 面板自带) -->
    <div class="flex items-center gap-2 border-b border-default px-3 py-2 shrink-0">
      <UIcon name="i-tabler-grid-dots" class="size-4 text-dimmed" />
      <span class="text-sm font-medium">{{ t('nodeExplorer.title') }}</span>
    </div>

    <div
      data-testid="node-library-search"
      class="shrink-0 border-b border-default bg-default px-3 py-3"
    >
      <UInput
        ref="searchInputRef"
        v-model="query"
        :placeholder="t('nodeExplorer.search_placeholder')"
        icon="i-tabler-search"
        size="sm"
        class="w-full"
      />
    </div>

    <div data-testid="node-library-scroll" class="flex-1 min-h-0 overflow-y-auto px-3 py-3">
      <!-- Tree body: per-group collapsible sections -->
      <div>
        <div v-if="filteredGroups.length === 0" class="text-center text-xs text-dimmed py-8 italic">
          {{ t('nodeExplorer.no_match') }}
        </div>
        <div v-else class="space-y-1">
          <div v-for="g in filteredGroups" :key="g.group">
            <button
              type="button"
              class="w-full flex items-center gap-2 px-2 py-1.5 hover:bg-elevated/30 rounded text-[12px] font-medium text-default"
              @click="toggleGroup(g.group)"
            >
              <UIcon
                :name="isExpanded(g.group) ? 'i-tabler-chevron-down' : 'i-tabler-chevron-right'"
                class="size-3.5"
              />
              <span>{{ groupLabelZh(g.group) }}</span>
              <span class="text-[10px] opacity-70">({{ g.specs.length }})</span>
            </button>
            <div
              v-show="isExpanded(g.group)"
              class="grid gap-1 pl-5 mt-1"
              style="grid-template-columns: repeat(auto-fill, minmax(160px, 1fr))"
            >
              <div
                v-for="spec in g.specs"
                :key="spec.kind"
                draggable="true"
                class="flex items-center gap-2 px-2 py-1 bg-elevated/30 hover:bg-elevated/60 rounded text-[11px] cursor-grab active:cursor-grabbing"
                :title="spec.description ? t(spec.description) : spec.kind"
                @click="onSelectKind(spec.kind)"
                @dragstart="(e) => startEditorDrag({ type: 'node-spec', kind: spec.kind }, e)"
              >
                <UIcon
                  v-if="spec.visual?.icon"
                  :name="spec.visual.icon"
                  class="size-3.5 shrink-0"
                  :class="nodeIconColor(spec)"
                />
                <span class="min-w-0 flex-1 truncate">{{
                  spec.labelZh ? t(spec.labelZh) : spec.kind
                }}</span>
                <span
                  v-for="badge in platformBadgesForTargets(spec.supportedTargets, {
                    isPureData: spec.isPureData,
                  })"
                  :key="badge.key"
                  class="shrink-0 rounded border px-1 py-0.5 text-[9px] leading-none"
                  :class="badge.class"
                >
                  {{ t(badge.labelKey) }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { allSpecs } from '@/components/containers/nodeRegistry/registry'
import type { NodeKindSpec } from '@/components/containers/nodeRegistry/index'
import { platformBadgesForTargets } from '@/components/containers/nodeRegistry/platformTargets'
import {
  ALL_NODE_GROUPS,
  nodeIconColor,
  groupLabelZh,
} from '@/composables/editor/useNodeGroupColor'
import { startEditorDrag } from '@/composables/editor/useEditorDragDrop'

const { t } = useI18n()

const EXPANDED_KEY = 'yotta.explorer.expanded'

const emit = defineEmits<{
  'pick-kind': [kind: string]
}>()

const query = ref('')
const searchInputRef = ref<any>(null)

// Expand state: Set<groupName>. Default = all known groups expanded.
const expandedGroups = ref<Set<string>>(new Set(ALL_NODE_GROUPS))

function loadExpandedFromStorage() {
  try {
    const raw = localStorage.getItem(EXPANDED_KEY)
    if (raw) {
      const arr = JSON.parse(raw) as string[]
      expandedGroups.value = new Set(arr)
    }
  } catch (_e) {}
}

function persistExpanded() {
  try {
    localStorage.setItem(EXPANDED_KEY, JSON.stringify([...expandedGroups.value]))
  } catch (_e) {}
}

loadExpandedFromStorage()

function toggleGroup(group: string) {
  if (expandedGroups.value.has(group)) {
    expandedGroups.value.delete(group)
  } else {
    expandedGroups.value.add(group)
  }
  // Force reactivity since Set mutation doesn't trigger
  expandedGroups.value = new Set(expandedGroups.value)
  persistExpanded()
}

function isExpanded(group: string): boolean {
  if (query.value.trim()) return true // Auto-expand all when searching
  return expandedGroups.value.has(group)
}

// 面板 mount 时 (rail 切到节点库 → v-if 新挂载) 聚焦搜索框 + 清 query.
onMounted(async () => {
  query.value = ''
  await nextTick()
  const el = searchInputRef.value?.$el as HTMLElement | undefined
  el?.querySelector('input')?.focus()
})

const filteredGroups = computed(() => {
  const q = query.value.toLowerCase().trim()
  const matched = allSpecs().filter((s: NodeKindSpec) => {
    if (s.excludeFromPalette) return false
    if (!q) return true
    const localizedLabel = s.labelZh ? t(s.labelZh) : ''
    const localizedDesc = s.description ? t(s.description) : ''
    const hay = `${s.kind} ${localizedLabel} ${localizedDesc}`.toLowerCase()
    return hay.includes(q)
  })
  const byGroup = new Map<string, NodeKindSpec[]>()
  for (const s of matched) {
    const g = s.group ?? 'misc'
    if (!byGroup.has(g)) byGroup.set(g, [])
    byGroup.get(g)!.push(s)
  }
  return Array.from(byGroup.entries())
    .map(([group, specs]) => ({ group, specs: specs.sort((a, b) => a.kind.localeCompare(b.kind)) }))
    .sort((a, b) => a.group.localeCompare(b.group))
})

function onSelectKind(kind: string) {
  emit('pick-kind', kind)
}
</script>
