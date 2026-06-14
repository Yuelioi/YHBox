<!-- 子图库资产停靠面板. 主操作 = 放进画布: 行可拖到画布(落松手处) / 双击插到中心.
     单击=选中(批量); 右键=快捷动作 + 详情(改名/描述/标签等低频 → 按需弹小 modal).
     在线 tab: 占位。 -->
<template>
  <div class="flex flex-col h-full min-h-0">
    <!-- 本地/在线 tab -->
    <div class="border-b border-default px-2 py-2 shrink-0">
      <UTabs v-model="activeTab" :items="tabItems" size="xs" :content="false" />
    </div>

    <!-- 在线: 占位 -->
    <div v-if="activeTab === 'online'" class="flex-1 flex flex-col items-center justify-center text-center py-16">
      <UIcon name="i-tabler-cloud" class="size-12 text-dimmed mb-3" />
      <h3 class="text-sm text-toned font-medium">{{ t('library.online.title') }}</h3>
      <p class="text-xs text-dimmed mt-2 max-w-xs">{{ t('library.online.desc') }}</p>
    </div>

    <!-- 本地: 列表占满 (详情走右键 → modal) -->
    <div v-else class="flex-1 min-h-0 overflow-hidden flex flex-col gap-2 p-3">
      <div class="flex items-center gap-3 shrink-0">
        <UInput
          ref="searchInputRef"
          v-model="query"
          :placeholder="t('library.explorer.search')"
          icon="i-tabler-search"
          size="sm"
          class="flex-1"
        />
        <USelect v-model="sortKey" :items="sortItems" size="xs" class="w-32" />
        <UButton
          size="xs"
          variant="ghost"
          :icon="sortDesc ? 'i-tabler-sort-descending' : 'i-tabler-sort-ascending'"
          :title="sortDesc ? t('library.explorer.sort_desc') : t('library.explorer.sort_asc')"
          @click="sortDesc = !sortDesc"
        />
      </div>

      <div class="flex items-center gap-2 shrink-0">
        <USelectMenu
          v-model="categoryFilter"
          :items="categoryFilterItems"
          value-key="id"
          size="xs"
          class="w-40"
        />
        <UInputMenu
          v-model="tagFilter"
          multiple
          :items="allTags"
          size="xs"
          :placeholder="t('library.explorer.filter_tags')"
          class="flex-1"
        />
      </div>

      <p class="text-[10px] text-dimmed px-1 shrink-0">{{ t('editor.dock.drag_hint') }}</p>

      <div class="flex-1 min-h-0 overflow-y-auto select-none">
        <div
          v-if="filteredItems.length === 0"
          class="text-center text-xs text-dimmed py-8 italic"
        >
          <span v-if="lib.loading">{{ t('library.loading') }}</span>
          <span v-else-if="lib.subgraphs.length === 0">{{ t('library.explorer.empty') }}</span>
          <span v-else>{{ t('library.explorer.no_match') }}</span>
        </div>

        <div v-else class="space-y-2">
          <template v-for="group in groupedItems" :key="group.category">
            <div class="text-[10px] font-semibold text-dimmed uppercase tracking-wider px-1 pt-2 pb-0.5">
              {{ group.category }}
            </div>
            <UContextMenu v-for="item in group.items" :key="item.id" :items="ctxMenuItems(item)">
              <div
                draggable="true"
                class="group rounded p-3 cursor-grab active:cursor-grabbing"
                :class="isSelected(item.id) ? 'bg-primary/15 ring-1 ring-inset ring-primary/50' : 'bg-elevated/30 hover:bg-elevated/60'"
                @click="onRowClick(item.id, $event)"
                @dblclick="onPick(item.id)"
                @contextmenu="selClick(item.id)"
                @dragstart="(e) => startEditorDrag({ type: 'library-subgraph', id: item.id }, e)"
              >
                <div class="flex items-start gap-2">
                  <span
                    class="mt-0.5 shrink-0 transition-opacity"
                    :class="isSelected(item.id) ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'"
                    @click.stop
                    @dblclick.stop
                    @dragstart.stop.prevent
                  >
                    <UCheckbox
                      :model-value="isSelected(item.id)"
                      size="sm"
                      @update:model-value="selClick(item.id, { ctrl: true })"
                    />
                  </span>
                  <UIcon name="i-tabler-package" class="size-4 text-primary mt-0.5 shrink-0" />
                  <div class="flex-1 min-w-0">
                    <div class="text-sm font-medium">{{ item.label }}</div>
                    <div v-if="item.description" class="text-[11px] text-dimmed mt-0.5 line-clamp-2">
                      {{ item.description }}
                    </div>
                    <div v-if="item.tags && item.tags.length > 0" class="flex flex-wrap gap-1 mt-1">
                      <span
                        v-for="tg in item.tags"
                        :key="tg"
                        class="px-1.5 py-0 bg-elevated/60 text-[9px] rounded text-dimmed"
                      >
                        {{ tg }}
                      </span>
                    </div>
                  </div>
                  <UButton
                    size="xs"
                    variant="ghost"
                    color="neutral"
                    icon="i-tabler-dots"
                    class="opacity-0 group-hover:opacity-100 shrink-0"
                    :aria-label="t('editor.dock.detail')"
                    @click.stop="openDetail(item.id)"
                    @dblclick.stop
                  />
                </div>
              </div>
            </UContextMenu>
          </template>
        </div>
      </div>

      <!-- 底部工具栏 (双态): 无选中 = 计数 + 分页; 有选中 = 批量操作 + 分页 -->
      <div class="flex items-center justify-between gap-3 pt-2 border-t border-default shrink-0">
        <div v-if="selected.size === 0" class="text-[11px] text-dimmed">
          {{ t('library.toolbar.total', { n: pageResult.total }) }}
        </div>
        <div v-else class="flex items-center gap-1.5 min-w-0">
          <span class="text-[11px] text-toned shrink-0">{{ t('library.batch.selected_n', { n: selected.size }) }}</span>
          <UDropdownMenu :items="batchMenuItems">
            <UButton size="xs" variant="soft" color="primary" icon="i-tabler-stack-2" trailing-icon="i-tabler-chevron-down">
              {{ t('library.batch.menu') }}
            </UButton>
          </UDropdownMenu>
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-tabler-x"
            :aria-label="t('library.batch.clear')"
            :title="t('library.batch.clear')"
            @click="selClear()"
          />
        </div>
        <div class="flex items-center gap-2 shrink-0">
          <UPagination
            v-if="pageResult.totalPages > 1"
            v-model:page="page"
            :total="pageResult.total"
            :items-per-page="pageSize"
            :sibling-count="1"
            size="xs"
          />
          <USelect v-model="pageSize" :items="pageSizeItems" size="xs" class="w-24" />
        </div>
      </div>
    </div>
  </div>

  <!-- 详情 (按需): 改名/描述/分类/标签/被引用统计/复制为新/删除 -->
  <BaseModal v-model:open="detailOpen" :title="t('editor.dock.detail')" icon="i-tabler-package" size="sm">
    <LibraryDetailPanel :sgID="detailId" @insert="onDetailInsert" />
  </BaseModal>

  <!-- 批量加标签 -->
  <BaseModal v-model:open="batchTagsOpen" :title="t('library.batch.add_tags_title')" icon="i-tabler-tags" size="md">
    <UInputMenu
      v-model="batchTags"
      multiple
      :create-item="'always'"
      :items="allTags"
      size="sm"
      :placeholder="t('library.batch.add_tags_placeholder')"
      @create="(v: string) => (batchTags = [...batchTags, v])"
    />
    <template #footer>
      <UButton variant="ghost" color="neutral" @click="batchTagsOpen = false">{{ t('common.cancel') }}</UButton>
      <UButton color="primary" :disabled="batchTags.length === 0" @click="onBatchAddTags">{{ t('library.batch.add_tags_apply') }}</UButton>
    </template>
  </BaseModal>

  <!-- 批量改分类 -->
  <BaseModal v-model:open="batchCategoryOpen" :title="t('library.batch.change_category_title')" icon="i-tabler-category" size="md">
    <UInputMenu
      v-model="batchCategory"
      :create-item="'always'"
      :items="allCategories"
      size="sm"
      :placeholder="t('library.batch.change_category_placeholder')"
      @create="(v: string) => (batchCategory = v)"
    />
    <template #footer>
      <UButton variant="ghost" color="neutral" @click="batchCategoryOpen = false">{{ t('common.cancel') }}</UButton>
      <UButton color="primary" @click="onBatchChangeCategory">{{ t('library.batch.change_category_apply') }}</UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { useLocalStorage } from '@vueuse/core'
import { useLibraryStore } from '@/stores/library'
import { useConfirm } from '@/composables/useConfirm'
import { useListSelection } from '@/composables/editor/useListSelection'
import { filterSubgraphs, groupByCategory, paginate } from '@/lib/libraryFilter'
import { startEditorDrag } from '@/composables/editor/useEditorDragDrop'
import BaseModal from '@/components/common/BaseModal.vue'
import LibraryDetailPanel from '@/components/containers/LibraryDetailPanel.vue'
import { backend, type Subgraph } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'

const { t } = useI18n()

const emit = defineEmits<{
  'pick-subgraph': [libraryID: string]
}>()

const query = ref('')
const searchInputRef = ref<any>(null)

// 排序 (镜像模板/clip 管理): 名称/创建时间/节点数 × 正逆序.
const sortKey = ref<'label' | 'createdAt' | 'nodes'>('label')
const sortDesc = ref(false)
const sortItems = computed(() => [
  { label: t('library.explorer.view_by_name'), value: 'label' },
  { label: t('library.explorer.view_by_created'), value: 'createdAt' },
  { label: t('library.explorer.view_by_nodes'), value: 'nodes' },
])

const lib = useLibraryStore()
const toast = useToast()
const { confirm } = useConfirm()

const activeTab = ref<'local' | 'online'>('local')
const tabItems = computed(() => [
  { label: t('library.explorer.tab_local'), value: 'local' },
  { label: t('library.explorer.tab_online'), value: 'online' },
])

async function refreshLibrary() {
  await lib.reload()
}

// ── 过滤 + 分组 ──
const categoryFilter = ref<string>('all')
const tagFilter = ref<string[]>([])

const allCategories = computed(() => {
  const set = new Set<string>()
  for (const sg of lib.subgraphs) if (sg.category) set.add(sg.category)
  return [...set].sort()
})
const allTags = computed(() => {
  const set = new Set<string>()
  for (const sg of lib.subgraphs) for (const tg of sg.tags ?? []) set.add(tg)
  return [...set].sort()
})

const categoryFilterItems = computed(() => [
  { label: t('library.explorer.filter_category_all'), id: 'all' },
  ...allCategories.value.map((c) => ({ label: c, id: `c:${c}` })),
  { label: t('library.explorer.uncategorized'), id: 'none' },
])

const filteredItems = computed<Subgraph[]>(() => {
  const arr = filterSubgraphs(lib.subgraphs, {
    query: query.value,
    category:
      categoryFilter.value === 'all' ? null : categoryFilter.value === 'none' ? '' : categoryFilter.value.slice(2),
    tags: tagFilter.value,
  })
  const sorted = [...arr]
  sorted.sort((a, b) => {
    let cmp = 0
    switch (sortKey.value) {
      case 'label': cmp = (a.label ?? '').localeCompare(b.label ?? ''); break
      case 'createdAt': cmp = (a.createdAt ?? '').localeCompare(b.createdAt ?? ''); break
      case 'nodes': cmp = (a.graph?.nodes?.length ?? 0) - (b.graph?.nodes?.length ?? 0); break
    }
    return sortDesc.value ? -cmp : cmp
  })
  return sorted
})

const page = ref(1)
const pageSize = useLocalStorage('library.pageSize', 50)
const pageSizeItems = computed(() => [20, 50, 100].map((n) => ({ label: t('library.toolbar.per_page', { n }), value: n })))

const pageResult = computed(() => paginate(filteredItems.value, page.value, pageSize.value))
const groupedItems = computed(() => groupByCategory(pageResult.value.pageItems, t('library.explorer.uncategorized')))

watch([query, categoryFilter, tagFilter, pageSize, sortKey, sortDesc], () => { page.value = 1 })
watch(() => pageResult.value.totalPages, (tp) => { if (page.value > tp) page.value = tp })

// 选中 (单击/Ctrl/Shift/勾选框) — 用于批量操作.
const visibleIds = computed(() => groupedItems.value.flatMap((g) => g.items.map((i) => i.id)))
const { selected, click: selClick, clear: selClear, isSelected } = useListSelection(visibleIds)

// 面板 mount → reload + 重置过滤 + 聚焦搜索.
onMounted(async () => {
  void refreshLibrary()
  query.value = ''
  categoryFilter.value = 'all'
  tagFilter.value = []
  page.value = 1
  selClear()
  await nextTick()
  const el = searchInputRef.value?.$el as HTMLElement | undefined
  el?.querySelector('input')?.focus()
})

function onRowClick(id: string, e: MouseEvent) {
  selClick(id, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey })
}

function onPick(libraryID: string) {
  emit('pick-subgraph', libraryID)
}

// ── 详情 (按需弹出) ──
const detailOpen = ref(false)
const detailId = ref<string | null>(null)
function openDetail(id: string) {
  detailId.value = id
  detailOpen.value = true
}
function onDetailInsert() {
  if (detailId.value) onPick(detailId.value)
  detailOpen.value = false
}

function ctxMenuItems(item: Subgraph) {
  return [
    [
      { label: t('library.explorer.insert'), icon: 'i-tabler-package-import', onSelect: () => onPick(item.id) },
      { label: t('editor.dock.detail'), icon: 'i-tabler-info-circle', onSelect: () => openDetail(item.id) },
      { label: t('library.card.duplicate'), icon: 'i-tabler-copy-plus', onSelect: () => onDuplicate(item) },
    ],
    [
      { label: t('library.card.copy_id'), icon: 'i-tabler-copy', onSelect: () => onCopyID(item) },
    ],
    [
      { label: t('library.card.delete'), icon: 'i-tabler-trash', color: 'error' as const, onSelect: () => onDelete(item) },
    ],
  ]
}

async function onCopyID(item: Subgraph) {
  try {
    await navigator.clipboard.writeText(item.id)
    toast.add({ title: t('toast.copy_id_success'), color: 'success', icon: 'i-tabler-check', duration: 1500 })
  } catch (e: any) {
    toast.add({ title: t('toast.copy_failed'), description: errorMessage(e), color: 'error' })
  }
}

// 复制为新子图 (fork): 想独立改不影响引用方时用。
async function onDuplicate(item: Subgraph) {
  const dup = await lib.duplicateSubgraph(item.id)
  if (dup) {
    toast.add({ title: t('library.card.duplicated', { name: dup.label }), color: 'success', icon: 'i-tabler-check' })
  }
}

// 删除安全: 先扫引用 — 被容器使用时警告里带"被 N 个容器使用", 确认才真删。
async function onDelete(item: Subgraph) {
  const refs = await lib.referrersOf(item.id)
  const useCount = lib.containerUseCount(refs)
  const name = item.label || item.id
  const desc = useCount > 0
    ? t('library.card.delete_confirm_referenced', { name, n: useCount })
    : t('library.card.delete_confirm_desc', { name })
  const yes = await confirm({
    title: t('library.card.delete_confirm_title'),
    description: desc,
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return
  const ok = await lib.deleteSubgraph(item.id)
  if (!ok) {
    toast.add({ title: t('toast.delete_failed'), color: 'error' })
  }
}

// ── 批量删除 ──
async function onBatchDelete() {
  const ids = [...selected.value]
  const referenced: string[] = []
  for (const id of ids) {
    const refs = await lib.referrersOf(id)
    if (lib.containerUseCount(refs) > 0) referenced.push(lib.byId(id)?.label || id)
  }
  const desc = referenced.length > 0
    ? t('library.batch.delete_confirm_referenced', { n: ids.length, m: referenced.length, names: referenced.join('、') })
    : t('library.batch.delete_confirm_desc', { n: ids.length })
  const yes = await confirm({
    title: t('library.batch.delete_confirm_title'),
    description: desc,
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return
  let failed = 0
  for (const id of ids) {
    if (!(await lib.deleteSubgraph(id))) failed++
  }
  if (failed > 0) {
    toast.add({ title: t('library.batch.partial_failed', { n: failed }), color: 'error' })
  }
  selClear()
}

// ── 批量加标签 ──
const batchTagsOpen = ref(false)
const batchTags = ref<string[]>([])

async function onBatchAddTags() {
  const add = batchTags.value.map((s) => s.trim()).filter(Boolean)
  if (add.length === 0) {
    batchTagsOpen.value = false
    return
  }
  const ids = [...selected.value]
  let failed = 0
  for (const id of ids) {
    const sg = lib.byId(id)
    if (!sg) { failed++; continue }
    const tags = [...new Set([...(sg.tags ?? []), ...add])]
    try {
      await backend.subgraphs.updateSilent(sg.id, JSON.stringify({ tags }), sg.rev)
    } catch {
      failed++
    }
  }
  await lib.reload()
  if (failed > 0) {
    toast.add({ title: t('library.batch.partial_failed', { n: failed }), color: 'error' })
  }
  batchTagsOpen.value = false
  batchTags.value = []
}

// 批量动作收进下拉.
const batchMenuItems = computed(() => [
  [
    { label: t('library.batch.add_tags'), icon: 'i-tabler-tags', onSelect: () => { batchTagsOpen.value = true } },
    { label: t('library.batch.change_category'), icon: 'i-tabler-category', onSelect: () => { batchCategoryOpen.value = true } },
  ],
  [
    { label: t('library.batch.delete'), icon: 'i-tabler-trash', color: 'error' as const, onSelect: () => { void onBatchDelete() } },
  ],
])

// ── 批量改分类 ──
const batchCategoryOpen = ref(false)
const batchCategory = ref('')

async function onBatchChangeCategory() {
  const target = batchCategory.value.trim()
  const ids = [...selected.value]
  let failed = 0
  for (const id of ids) {
    const sg = lib.byId(id)
    if (!sg) { failed++; continue }
    try {
      await backend.subgraphs.updateSilent(sg.id, JSON.stringify({ category: target }), sg.rev)
    } catch {
      failed++
    }
  }
  await lib.reload()
  if (failed > 0) {
    toast.add({ title: t('library.batch.partial_failed', { n: failed }), color: 'error' })
  }
  batchCategoryOpen.value = false
  batchCategory.value = ''
}
</script>
