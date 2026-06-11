<template>
  <div class="flex flex-col h-full text-xs">
    <UTabs v-model="activeTab" :items="paletteTabs" size="xs" :ui="{ list: 'h-8 shrink-0' }" />

    <div class="flex-1 overflow-y-auto pt-2 space-y-2">
      <!-- 节点 tab -->
      <template v-if="activeTab === 'nodes'">
        <div class="px-1 pb-1 sticky top-0 bg-default z-10">
          <UInput
            v-model="nodeSearch"
            :placeholder="t('nodePalette.search_node')"
            icon="i-tabler-search"
            size="xs"
          />
        </div>
        <div v-for="group in filteredGroups" :key="group.label">
          <div class="font-medium text-toned mb-1">{{ group.label }}</div>
          <div class="space-y-0.5">
            <button
              v-for="n in group.items"
              :key="n.kind"
              type="button"
              class="w-full text-left px-2 py-1 rounded hover:bg-elevated/40 text-default transition-colors cursor-grab active:cursor-grabbing"
              draggable="true"
              @dragstart="onNodeDragStart($event, n.kind)"
              @click="$emit('add', n.kind)"
            >
              <UIcon :name="n.icon" class="size-3.5 mr-1.5 inline text-dimmed" />
              {{ n.label }}
            </button>
          </div>
        </div>
        <p v-if="filteredGroups.length === 0" class="text-[11px] text-dimmed text-center py-4">
          {{ t('nodePalette.no_match_node') }}
        </p>
      </template>

      <!-- 库 tab：库子图 mini 列表 -->
      <template v-else-if="activeTab === 'library'">
        <div class="px-1 pb-1">
          <UInput
            v-model="libSearch"
            :placeholder="t('nodePalette.search_library')"
            icon="i-tabler-search"
            size="xs"
          />
        </div>

        <p v-if="libraryStore.subgraphs.length === 0" class="text-[11px] text-dimmed text-center py-4">
          {{ t('nodePalette.library_empty') }}<br />
          <a class="text-primary cursor-pointer hover:underline" @click="goLibrary">{{ t('nodePalette.go_library_manage') }}</a>
        </p>
        <p
          v-else-if="filteredLibrary.length === 0"
          class="text-[11px] text-dimmed text-center py-4"
        >
          {{ t('nodePalette.no_match_subgraph') }}
        </p>

        <div v-for="sg in filteredLibrary" :key="sg.id" class="px-0.5">
          <UContextMenu :items="ctxMenuItemsFor(sg)">
            <div
              draggable="true"
              class="rounded-md border border-default/60 bg-elevated/30 p-1.5 cursor-grab hover:border-primary/40 transition-colors"
              :title="tooltipFor(sg)"
              @dragstart="onLibraryDragStart($event, sg)"
              @dblclick="goLibrary"
            >
              <div class="flex items-center gap-1.5">
                <UIcon name="i-tabler-package" class="size-3 text-fuchsia-300 shrink-0" />
                <span class="text-[11px] truncate flex-1">{{ sg.label || sg.id }}</span>
                <UBadge size="xs" variant="subtle" color="neutral" class="shrink-0">
                  {{ sg.graph?.nodes?.length ?? 0 }}
                </UBadge>
              </div>
              <div v-if="(sg.tags ?? []).length > 0" class="flex flex-wrap gap-0.5 mt-1">
                <UBadge
                  v-for="t in (sg.tags ?? []).slice(0, 3)"
                  :key="t"
                  size="xs"
                  variant="subtle"
                  class="text-[9px] py-0 px-1"
                >
                  {{ t }}
                </UBadge>
                <UBadge
                  v-if="(sg.tags ?? []).length > 3"
                  size="xs"
                  variant="ghost"
                  class="text-[9px] py-0 px-1"
                >
                  +{{ (sg.tags ?? []).length - 3 }}
                </UBadge>
              </div>
            </div>
          </UContextMenu>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useLibraryStore } from '@/stores/library'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { useToast } from '@nuxt/ui/composables'
import { backend, type Subgraph } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'
import { allSpecs } from '@/components/containers/nodeRegistry/registry'
import type { NodeKindSpec, NodeGroup } from '@/components/containers/nodeRegistry/index'

const { t } = useI18n()

defineEmits<{ add: [kind: string] }>()

const libraryStore = useLibraryStore()
const editorStore = useContainerEditorStore()
const router = useRouter()
const toast = useToast()

const paletteTabs = computed(() => [
  { label: t('nodePalette.tab_nodes'), value: 'nodes' },
  { label: t('nodePalette.tab_library'), value: 'library' },
])
const activeTab = ref('nodes')
const libSearch = ref('')
// v4 B10: search filter for built-in node palette (matches kind or label substring, case-insensitive)
const nodeSearch = ref('')

const filteredLibrary = computed(() => {
  const q = libSearch.value.trim().toLowerCase()
  let list = libraryStore.subgraphs.slice()
  if (q) {
    list = list.filter(
      (sg) =>
        (sg.label ?? '').toLowerCase().includes(q) ||
        (sg.id ?? '').toLowerCase().includes(q) ||
        (sg.description ?? '').toLowerCase().includes(q) ||
        (sg.tags ?? []).some((t) => t.toLowerCase().includes(q)),
    )
  }
  return list
})

function tooltipFor(sg: Subgraph): string {
  const parts = [sg.label || sg.id]
  if (sg.description) parts.push(sg.description)
  if ((sg.tags ?? []).length > 0) parts.push('#' + (sg.tags ?? []).join(' #'))
  return parts.join('\n')
}

function ctxMenuItemsFor(sg: Subgraph) {
  return [
    [
      {
        label: t('nodePalette.ctx_import_current'),
        icon: 'i-tabler-arrow-bar-to-down',
        disabled: !editorStore.activeContainerID,
        onSelect: () => onImport(sg),
      },
    ],
    [
      {
        label: t('nodePalette.ctx_view_in_library'),
        icon: 'i-tabler-external-link',
        onSelect: () => goLibrary(),
      },
      {
        label: t('nodePalette.ctx_copy_id'),
        icon: 'i-tabler-copy',
        onSelect: () => onCopyID(sg.id),
      },
    ],
  ]
}

async function onImport(sg: Subgraph) {
  if (!editorStore.activeContainerID) {
    toast.add({ title: t('nodePalette.toast_no_active_editor'), color: 'warning' })
    return
  }
  try {
    await libraryStore.importEnsuringGlobals(sg.id, editorStore.activeContainerID)
  } catch (e: any) {
    toast.add({ title: t('toast.import_failed'), description: errorMessage(e), color: 'error' })
  }
}

async function onCopyID(id: string) {
  try {
    await navigator.clipboard.writeText(id)
    toast.add({ title: t('toast.copy_id_success'), color: 'success', icon: 'i-tabler-check', duration: 1500 })
  } catch (e: any) {
    toast.add({ title: t('toast.copy_failed'), description: errorMessage(e), color: 'error' })
  }
}

function onNodeDragStart(e: DragEvent, kind: string) {
  if (!e.dataTransfer) return
  e.dataTransfer.effectAllowed = 'copy'
  e.dataTransfer.setData('application/x-yotta-node', kind)
}

function onLibraryDragStart(e: DragEvent, sg: Subgraph) {
  if (!e.dataTransfer) return
  e.dataTransfer.effectAllowed = 'copy'
  e.dataTransfer.setData(
    'application/yotta-library-item',
    JSON.stringify({ kind: 'subgraph', id: sg.id }),
  )
}

function goLibrary() {
  void router.push('/library')
}

onMounted(() => {
  void libraryStore.reload()
})

// v4 B5: palette sections derived from nodeRegistry. Adding a kind in
// nodeRegistry/specs/<group>.ts auto-populates the palette — no edit here needed.
// GROUP_LABEL maps the 6 spec groups to Chinese palette section headers.
// Order in GROUP_LABEL determines section render order.
const GROUP_LABEL = computed<Record<NodeGroup, string>>(() => ({
  control: t('nodeGroup.control'),
  variables: t('nodePalette.group_variables_data'),
  purefunc: t('nodePalette.group_purefunc'),
  detect: t('nodeGroup.detect'),
  input: t('nodeGroup.input'),
  system: t('nodeGroup.system'),
  io: 'IO',
  stopwatch: t('nodeGroup.stopwatch'),
  mock: t('nodeGroup.mock'),
  test: t('nodeGroup.test'),
  event: t('nodeGroup.event'),
  random: t('nodeGroup.random'),
  list: t('nodeGroup.list'),
}))

interface PaletteItem {
  kind: string
  icon: string
  label: string
}
interface PaletteGroup {
  label: string
  items: PaletteItem[]
}

const KINDS_BY_GROUP = computed<Record<NodeGroup, PaletteGroup>>(() => {
  const labels = GROUP_LABEL.value
  const groups: Record<NodeGroup, PaletteGroup> = {
    control: { label: labels.control, items: [] },
    variables: { label: labels.variables, items: [] },
    purefunc: { label: labels.purefunc, items: [] },
    detect: { label: labels.detect, items: [] },
    input: { label: labels.input, items: [] },
    system: { label: labels.system, items: [] },
    io: { label: labels.io, items: [] },
    stopwatch: { label: labels.stopwatch, items: [] },
    mock: { label: labels.mock, items: [] },
    test: { label: labels.test, items: [] },
    event: { label: labels.event, items: [] },
    random: { label: labels.random, items: [] },
    list: { label: labels.list, items: [] },
  }
  for (const s of allSpecs() as NodeKindSpec[]) {
    if (s.excludeFromPalette) continue // 不入面板的节点 (markers/visual-only — CommentBox 例外, adapter 已放行)
    const g = groups[s.group]
    if (!g) continue
    g.items.push({ kind: s.kind, icon: s.visual.icon, label: s.labelZh ? t(s.labelZh) : s.kind })
  }
  for (const g of Object.values(groups)) {
    g.items.sort((a, b) => a.label.localeCompare(b.label, 'zh'))
  }
  return groups
})

// v4 B10: search-aware computed groups. Empty query → all groups; non-empty → only
// groups with at least one matching item, items filtered by substring on kind+label.
const filteredGroups = computed<PaletteGroup[]>(() => {
  const all = Object.values(KINDS_BY_GROUP.value).filter((g) => g.items.length > 0)
  const q = nodeSearch.value.trim().toLowerCase()
  if (!q) return all
  return all
    .map((g) => ({
      ...g,
      items: g.items.filter(
        (n) => n.kind.toLowerCase().includes(q) || n.label.toLowerCase().includes(q),
      ),
    }))
    .filter((g) => g.items.length > 0)
})
</script>
