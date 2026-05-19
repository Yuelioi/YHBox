<template>
  <div class="flex flex-col h-full text-xs">
    <UTabs v-model="activeTab" :items="paletteTabs" size="xs" :ui="{ list: 'h-8 shrink-0' }" />

    <div class="flex-1 overflow-y-auto pt-2 space-y-2">
      <!-- 节点 tab -->
      <template v-if="activeTab === 'nodes'">
        <div class="px-1 pb-1 sticky top-0 bg-default z-10">
          <UInput
            v-model="nodeSearch"
            placeholder="搜索节点 (kind 或 中文名)..."
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
          没匹配的节点
        </p>
      </template>

      <!-- 库 tab：库子图 mini 列表 -->
      <template v-else-if="activeTab === 'library'">
        <div class="px-1 pb-1">
          <UInput
            v-model="libSearch"
            placeholder="搜索库子图..."
            icon="i-tabler-search"
            size="xs"
          />
        </div>

        <p v-if="libraryStore.subgraphs.length === 0" class="text-[11px] text-dimmed text-center py-4">
          库还是空的<br />
          <a class="text-primary cursor-pointer hover:underline" @click="goLibrary">去库管理</a>
        </p>
        <p
          v-else-if="filteredLibrary.length === 0"
          class="text-[11px] text-dimmed text-center py-4"
        >
          没有匹配的子图
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
import { useRouter } from 'vue-router'
import { useLibraryStore } from '@/stores/library'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { useToast } from '@nuxt/ui/composables'
import { backend, type LibrarySubgraph } from '@/lib/backend'
import { allSpecs } from '@/components/containers/nodeRegistry/registry'
import type { NodeKindSpec, NodeGroup } from '@/components/containers/nodeRegistry/index'

defineEmits<{ add: [kind: string] }>()

const libraryStore = useLibraryStore()
const editorStore = useContainerEditorStore()
const router = useRouter()
const toast = useToast()

const paletteTabs = [
  { label: '节点', value: 'nodes' },
  { label: '库', value: 'library' },
]
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

function tooltipFor(sg: LibrarySubgraph): string {
  const parts = [sg.label || sg.id]
  if (sg.description) parts.push(sg.description)
  if ((sg.tags ?? []).length > 0) parts.push('#' + (sg.tags ?? []).join(' #'))
  return parts.join('\n')
}

function ctxMenuItemsFor(sg: LibrarySubgraph) {
  return [
    [
      {
        label: '导入到当前容器',
        icon: 'i-tabler-arrow-bar-to-down',
        disabled: !editorStore.activeContainerID,
        onSelect: () => onImport(sg),
      },
    ],
    [
      {
        label: '在库管理中查看',
        icon: 'i-tabler-external-link',
        onSelect: () => goLibrary(),
      },
      {
        label: '复制 ID',
        icon: 'i-tabler-copy',
        onSelect: () => onCopyID(sg.id),
      },
    ],
  ]
}

async function onImport(sg: LibrarySubgraph) {
  if (!editorStore.activeContainerID) {
    toast.add({ title: '当前没有打开的容器编辑器', color: 'warning' })
    return
  }
  try {
    const r = (await backend.library.copyToContainer(sg.id, editorStore.activeContainerID)) as any
    toast.add({
      title: `已导入子图: ${r?.label ?? r?.id ?? ''}`,
      color: 'success',
      icon: 'i-tabler-check',
    })
  } catch (e: any) {
    toast.add({ title: '导入失败', description: String(e?.message ?? e), color: 'error' })
  }
}

async function onCopyID(id: string) {
  try {
    await navigator.clipboard.writeText(id)
    toast.add({ title: '已复制 ID', color: 'success', icon: 'i-tabler-check' })
  } catch (e: any) {
    toast.add({ title: '复制失败', description: String(e?.message ?? e), color: 'error' })
  }
}

function onNodeDragStart(e: DragEvent, kind: string) {
  if (!e.dataTransfer) return
  e.dataTransfer.effectAllowed = 'copy'
  e.dataTransfer.setData('application/x-yhbox-node', kind)
}

function onLibraryDragStart(e: DragEvent, sg: LibrarySubgraph) {
  if (!e.dataTransfer) return
  e.dataTransfer.effectAllowed = 'copy'
  e.dataTransfer.setData(
    'application/yhfish-library-item',
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
const GROUP_LABEL: Record<NodeGroup, string> = {
  control: '控制流',
  variables: '变量 / 数据',
  purefunc: '纯函数',
  detect: '检测',
  input: '输入',
  system: '系统',
}

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
  const groups: Record<NodeGroup, PaletteGroup> = {
    control: { label: GROUP_LABEL.control, items: [] },
    variables: { label: GROUP_LABEL.variables, items: [] },
    purefunc: { label: GROUP_LABEL.purefunc, items: [] },
    detect: { label: GROUP_LABEL.detect, items: [] },
    input: { label: GROUP_LABEL.input, items: [] },
    system: { label: GROUP_LABEL.system, items: [] },
  }
  for (const s of allSpecs() as NodeKindSpec[]) {
    if (s.isVisualOnly) continue // CommentBox — not draggable from palette
    if (s.excludeFromPalette) continue // SubgraphInput/Output/CollapsedNode — created via dedicated UI
    const g = groups[s.group]
    if (!g) continue
    g.items.push({ kind: s.kind, icon: s.visual.icon, label: s.labelZh })
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
