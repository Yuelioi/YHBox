<!-- Houdini-style collapsible tree node browser.
     入口: toolbar 📐 / Tab 键 (canvas focused). -->
<template>
  <UModal v-model:open="modelOpen" :ui="{ content: 'sm:max-w-5xl' }">
    <template #content>
      <div class="p-5 bg-default space-y-3" style="min-height: 60vh; max-height: 80vh; display: flex; flex-direction: column;">
        <!-- Header: title + search -->
        <div class="flex items-center gap-3 shrink-0">
          <UIcon name="i-tabler-grid-dots" class="size-5 text-primary" />
          <h3 class="text-sm font-medium">{{ t('nodeExplorer.title') }}</h3>
          <UInput
            ref="searchInputRef"
            v-model="query"
            :placeholder="t('nodeExplorer.search_placeholder')"
            icon="i-tabler-search"
            size="sm"
            class="flex-1"
            @keydown.escape="onEsc"
          />
          <span class="text-[10px] text-dimmed">{{ t('nodeExplorer.esc_tab_close') }}</span>
        </div>

        <!-- Tree body: per-group collapsible sections -->
        <div class="flex-1 overflow-y-auto pr-2">
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
                <UIcon :name="isExpanded(g.group) ? 'i-tabler-chevron-down' : 'i-tabler-chevron-right'" class="size-3.5" />
                <span>{{ groupLabelZh(g.group) }}</span>
                <span class="text-[10px] opacity-70">({{ g.specs.length }})</span>
              </button>
              <div v-show="isExpanded(g.group)" class="grid gap-1 pl-5 mt-1" style="grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));">
                <div
                  v-for="spec in g.specs"
                  :key="spec.kind"
                  class="flex items-center gap-2 px-2 py-1 bg-elevated/30 hover:bg-elevated/60 rounded text-[11px] cursor-pointer"
                  :title="spec.description ? t(spec.description) : spec.kind"
                  @click="onSelectKind(spec.kind)"
                >
                  <UIcon v-if="spec.visual?.icon" :name="spec.visual.icon" class="size-3.5 shrink-0" :class="nodeIconColor(spec)" />
                  <span class="flex-1 truncate">{{ spec.labelZh ? t(spec.labelZh) : spec.kind }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { allSpecs } from '@/components/containers/nodeRegistry/registry'
import type { NodeKindSpec } from '@/components/containers/nodeRegistry/index'
import { ALL_NODE_GROUPS, nodeIconColor, groupLabelZh } from '@/composables/editor/useNodeGroupColor'
import { useDialogOpen } from '@/composables/editor/useDialogOpen'
import { useAutoFocusOnOpen } from '@/composables/editor/useAutoFocusOnOpen'

const { t } = useI18n()

const EXPANDED_KEY = 'yotta.explorer.expanded'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{
  'update:open': [v: boolean]
  'pick-kind': [kind: string]
}>()

const modelOpen = useDialogOpen(props, emit)

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
  if (query.value.trim()) return true  // Auto-expand all when searching
  return expandedGroups.value.has(group)
}

// On open: focus search + clear query
useAutoFocusOnOpen(modelOpen, searchInputRef, {
  onOpen: () => { query.value = '' },
})

const filteredGroups = computed(() => {
  const q = query.value.toLowerCase().trim()
  const matched = allSpecs().filter((s: NodeKindSpec) => {
    if (s.isVisualOnly) return false
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
  modelOpen.value = false
}

function onEsc() {
  modelOpen.value = false
}
</script>
