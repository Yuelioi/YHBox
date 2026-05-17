<!--
LibraryView 主程序「库」视图 — UE Content Browser 风格：
  ┌─────────────────────────────────────┬──────────────────────┐
  │ 顶部 Tabs（本机 / 在线）+ actions    │ Detail Panel         │
  ├─────────────────────────────────────┤ 单击卡片预览 / 编辑  │
  │ Type tabs（全部 / 子图 / 模板 / 片段）│ subgraph / template  │
  │ Toolbar：搜索 + 视图模式 + 选择批量   │ 受限只读（无 RPC）   │
  │ Tag filter（仅 subgraphs/templates）  │ clip 全字段可编辑   │
  │ 网格 / 列表（双击 = 进入；右键 = 菜单）│                      │
  └─────────────────────────────────────┴──────────────────────┘
-->
<template>
  <div class="flex flex-col h-full bg-default text-default">
    <TabToolbar :tabs="sourceTabs" v-model:activeTab="sourceTab">
      <template #actions>
        <UButton
          v-if="sourceTab === 'local'"
          size="sm"
          :variant="batch.enabled.value ? 'solid' : 'soft'"
          color="neutral"
          icon="i-tabler-checks"
          @click="batch.toggleMode()"
        >
          {{ batch.enabled.value ? '退出选择' : '选择' }}
        </UButton>
        <UButton
          v-if="batch.enabled.value && batch.count.value > 0"
          size="sm"
          color="error"
          icon="i-tabler-trash"
          @click="onBatchDelete"
        >
          删除 ({{ batch.count.value }})
        </UButton>
        <UButton size="sm" variant="ghost" color="neutral" icon="i-tabler-refresh" @click="reload">
          刷新
        </UButton>
      </template>
    </TabToolbar>

    <div v-if="sourceTab === 'local'" class="flex-1 flex overflow-hidden min-h-0">
      <!-- 主区：tabs + 工具栏 + 网格/列表 -->
      <div class="flex-1 flex flex-col overflow-hidden min-h-0">
        <div class="px-4 py-2 border-b border-default">
          <UTabs :items="typeTabs" v-model="typeTab" size="xs" />
        </div>

        <!-- 工具栏 -->
        <div class="px-4 pt-3 pb-2 flex items-center gap-2 shrink-0">
          <UInput
            v-model="search"
            :placeholder="typeTab === 'clips' ? '搜索片段...' : '搜索...'"
            icon="i-tabler-search"
            size="sm"
            class="flex-1"
          />
          <UButtonGroup size="sm">
            <UButton
              :variant="viewMode === 'grid' ? 'solid' : 'soft'"
              color="neutral"
              icon="i-tabler-layout-grid"
              title="网格视图"
              @click="setViewMode('grid')"
            />
            <UButton
              :variant="viewMode === 'list' ? 'solid' : 'soft'"
              color="neutral"
              icon="i-tabler-list"
              title="列表视图"
              @click="setViewMode('list')"
            />
          </UButtonGroup>
        </div>

        <!-- Tag filter（subgraphs / templates / 全部）-->
        <div v-if="typeTab !== 'clips'" class="px-4 pb-2">
          <LibraryTagFilter v-model:selected-tags="selectedTags" :tags="allTagsByCount" />
        </div>

        <!-- 列表本体 -->
        <div class="flex-1 overflow-y-auto px-4 pb-4">
          <!-- 非 clips：subgraphs / templates / 全部 -->
          <template v-if="typeTab !== 'clips'">
            <div
              v-if="filtered.length > 0"
              :class="viewMode === 'grid' ? 'grid grid-cols-3 gap-3' : 'flex flex-col gap-1.5'"
            >
              <div
                v-for="item in filtered"
                :key="item.kind + '-' + itemID(item)"
                class="relative"
              >
                <UCheckbox
                  v-if="batch.enabled.value"
                  :model-value="batch.isSelected(itemKey(item))"
                  size="sm"
                  class="absolute top-2 left-2 z-10"
                  @click.stop
                  @update:model-value="batch.toggle(itemKey(item))"
                />
                <LibraryCard
                  :item="item"
                  :view-mode="viewMode"
                  :selected="isSelected(item)"
                  @select="onSelectItem"
                  @open="onOpenItem"
                  @import-to-active="onImportToActive"
                />
              </div>
            </div>
            <p v-else class="text-xs text-dimmed text-center py-8">
              {{
                libraryStore.loading
                  ? '加载中...'
                  : search || selectedTags.length > 0
                    ? '没有匹配的项'
                    : '库还是空的，从容器编辑器把子图保存到库即可'
              }}
            </p>
          </template>

          <!-- clips tab -->
          <template v-else>
            <div
              v-if="filteredClips.length > 0"
              :class="viewMode === 'grid' ? 'grid grid-cols-3 gap-3' : 'flex flex-col gap-1.5'"
            >
              <LibraryClipCard
                v-for="c in filteredClips"
                :key="c.id"
                :clip="c"
                :view-mode="viewMode"
                :selected="selection?.kind === 'clip' && selection.payload.id === c.id"
                @select="onSelectClip"
              />
            </div>
            <p v-else class="text-xs text-dimmed text-center py-8">
              {{
                clipsStore.loading
                  ? '加载中...'
                  : search
                    ? '没有匹配的片段'
                    : '没有录制片段，从容器编辑器录制 InputClip 即可'
              }}
            </p>
          </template>
        </div>
      </div>

      <!-- 右侧 detail panel -->
      <LibraryDetailPanel
        :selection="selection"
        @import-subgraph="onImportSubgraphByID"
        @cleared="selection = null"
      />
    </div>

    <ComingSoonSection v-else-if="sourceTab === 'online'" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import TabToolbar from '@/components/common/TabToolbar.vue'
import LibraryCard from '@/components/library/LibraryCard.vue'
import LibraryClipCard from '@/components/library/LibraryClipCard.vue'
import LibraryTagFilter from '@/components/library/LibraryTagFilter.vue'
import LibraryDetailPanel, {
  type LibrarySelection,
} from '@/components/library/LibraryDetailPanel.vue'
import ComingSoonSection from '@/components/library/ComingSoonSection.vue'
import { useLibraryStore } from '@/stores/library'
import { useClipsStore, type ClipSummary } from '@/stores/clips'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { useToast } from '@nuxt/ui/composables'
import { backend, type LibraryItem } from '@/lib/backend'
import { useBatchSelect } from '@/composables/useBatchSelect'
import { useConfirm } from '@/composables/useConfirm'
import { useLibraryViewMode } from '@/composables/useLibraryViewMode'

const libraryStore = useLibraryStore()
const clipsStore = useClipsStore()
const editorStore = useContainerEditorStore()
const toast = useToast()

const batch = useBatchSelect()
const { confirm } = useConfirm()
const { mode: viewMode, set: setViewMode } = useLibraryViewMode()

// ─── selection ───────────────────────────────────────────────────────────
// 联合 — detail panel 走它
const selection = ref<LibrarySelection | null>(null)

function isSelected(item: LibraryItem): boolean {
  if (!selection.value) return false
  if (selection.value.kind === 'subgraph' && item.kind === 'subgraph') {
    return selection.value.payload.id === item.id
  }
  if (selection.value.kind === 'template' && item.kind === 'template') {
    return selection.value.payload.key === item.key
  }
  return false
}

function onSelectItem(item: LibraryItem) {
  if (batch.enabled.value) {
    batch.toggle(itemKey(item))
    return
  }
  if (item.kind === 'subgraph') selection.value = { kind: 'subgraph', payload: item }
  else selection.value = { kind: 'template', payload: item }
}

function onSelectClip(c: ClipSummary) {
  selection.value = { kind: 'clip', payload: c }
}

function onOpenItem(item: LibraryItem) {
  if (item.kind === 'subgraph') {
    // 暂未支持直接进库 subgraph 编辑器（需先 CopyToContainer）。
    // 提示用户走"导入到当前容器"或在已打开的编辑器里直接拖入。
    toast.add({
      title: '想编辑库子图？',
      description: '先「导入到当前容器」，在编辑器内修改，再发布回库。',
      color: 'info',
      icon: 'i-tabler-info-circle',
    })
  } else {
    onSelectItem(item)
  }
}

// 当 store 列表 reload 后, 同步 selection.payload 指向新对象（避免 stale data）
watch(
  () => [libraryStore.subgraphs, libraryStore.templates, clipsStore.clips] as const,
  () => {
    const cur = selection.value
    if (!cur) return
    if (cur.kind === 'subgraph') {
      const found = libraryStore.subgraphs.find((s) => s.id === cur.payload.id)
      selection.value = found ? { kind: 'subgraph', payload: found } : null
    } else if (cur.kind === 'template') {
      const found = libraryStore.templates.find((t) => t.key === cur.payload.key)
      selection.value = found ? { kind: 'template', payload: found } : null
    } else {
      const found = clipsStore.clips.find((c) => c.id === cur.payload.id)
      selection.value = found ? { kind: 'clip', payload: found } : null
    }
  },
  { deep: true },
)

// ─── batch ───────────────────────────────────────────────────────────────
function itemKey(item: LibraryItem): string {
  return `${item.kind}-${itemID(item)}`
}
function itemID(item: LibraryItem): string {
  return item.kind === 'subgraph' ? item.id : item.key
}

async function onBatchDelete() {
  const sgIDs: string[] = []
  const tplKeys: string[] = []
  for (const item of filtered.value) {
    if (!batch.isSelected(itemKey(item))) continue
    if (item.kind === 'subgraph') sgIDs.push(item.id)
    else tplKeys.push(item.key)
  }
  const total = sgIDs.length + tplKeys.length
  if (total === 0) return
  const yes = await confirm({
    title: '批量删除库内容',
    description: `确认删除 ${sgIDs.length} 个子图 + ${tplKeys.length} 个模板？此操作不可恢复。`,
    color: 'error',
    confirmText: '删除',
  })
  if (yes !== true) return
  try {
    if (sgIDs.length > 0) await backend.library.deleteSubgraphMany(sgIDs)
    if (tplKeys.length > 0) await backend.library.deleteTemplateMany(tplKeys)
    toast.add({ title: `已删除 ${total} 项`, color: 'success' })
    batch.clear()
    batch.disable()
    await libraryStore.reload()
  } catch (e: any) {
    toast.add({ title: '批量删除部分失败', description: String(e?.message ?? e), color: 'error' })
  }
}

// ─── import ──────────────────────────────────────────────────────────────
async function onImportToActive(item: LibraryItem) {
  if (item.kind !== 'subgraph') return
  await doImportSubgraph(item.id)
}
async function onImportSubgraphByID(id: string) {
  await doImportSubgraph(id)
}
async function doImportSubgraph(id: string) {
  if (!editorStore.activeContainerID) {
    toast.add({ title: '当前没有打开的容器编辑器', color: 'warning' })
    return
  }
  try {
    const r = (await backend.library.copyToContainer(id, editorStore.activeContainerID)) as any
    toast.add({
      title: `已导入子图: ${r?.label ?? r?.id ?? ''}`,
      color: 'success',
      icon: 'i-tabler-check',
    })
  } catch (e: any) {
    toast.add({ title: '导入失败', description: String(e?.message ?? e), color: 'error' })
  }
}

// ─── tabs / 筛选 ─────────────────────────────────────────────────────────
const sourceTabs = [
  { label: '本机', value: 'local' },
  { label: '在线（即将上线）', value: 'online' },
]
const sourceTab = ref('local')

const typeTabs = [
  { label: '全部', value: 'all' },
  { label: '子图', value: 'subgraphs' },
  { label: '模板', value: 'templates' },
  { label: '录制片段', value: 'clips' },
]
const typeTab = ref('all')

const search = ref('')
const selectedTags = ref<string[]>([])

// 当 typeTab 变化时清 selection / batch（语义边界）
watch(typeTab, () => {
  selection.value = null
  batch.clear()
})

const filteredClips = computed(() => {
  let list = clipsStore.clips.slice()
  if (search.value) {
    const q = search.value.toLowerCase()
    list = list.filter(
      (c) =>
        (c.label ?? '').toLowerCase().includes(q) ||
        (c.id ?? '').toLowerCase().includes(q) ||
        (c.description ?? '').toLowerCase().includes(q) ||
        (c.tags ?? []).some((t) => t.toLowerCase().includes(q)),
    )
  }
  return list
})

const filtered = computed<LibraryItem[]>(() => {
  let list: LibraryItem[] = []
  if (typeTab.value === 'all' || typeTab.value === 'subgraphs') {
    list.push(
      ...libraryStore.subgraphs.map((s) => ({ kind: 'subgraph', ...s }) as LibraryItem),
    )
  }
  if (typeTab.value === 'all' || typeTab.value === 'templates') {
    list.push(
      ...libraryStore.templates.map((t) => ({ kind: 'template', ...t }) as LibraryItem),
    )
  }
  if (search.value) {
    const q = search.value.toLowerCase()
    list = list.filter((x) => {
      const name =
        x.kind === 'subgraph' ? (x.label ?? '') : (x.meta?.name ?? '')
      const id = x.kind === 'subgraph' ? x.id : x.key
      return (
        name.toLowerCase().includes(q) ||
        id.toLowerCase().includes(q)
      )
    })
  }
  if (selectedTags.value.length > 0) {
    list = list.filter((x) => {
      const tags: string[] = x.kind === 'subgraph' ? (x.tags ?? []) : (x.meta?.tags ?? [])
      return selectedTags.value.every((t) => tags.includes(t))
    })
  }
  return list
})

const allTagsByCount = computed(() => {
  const counts: Record<string, number> = {}
  for (const s of libraryStore.subgraphs) for (const t of s.tags ?? []) counts[t] = (counts[t] ?? 0) + 1
  for (const t of libraryStore.templates)
    for (const tag of t.meta?.tags ?? []) counts[tag] = (counts[tag] ?? 0) + 1
  return Object.entries(counts)
    .sort((a, b) => b[1] - a[1])
    .map(([tag, c]) => ({ tag, count: c }))
})

async function reload() {
  await libraryStore.reload()
  await clipsStore.refresh()
}

onMounted(() => {
  void reload()
  clipsStore.listen()
})
</script>
