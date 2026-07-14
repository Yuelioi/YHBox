<!-- 子图库资产停靠面板. 主操作 = 放进画布: 行可拖到画布(落松手处) / 双击插到中心.
     单击=选中(批量); 右键=快捷动作 + 详情(改名/描述/标签等低频 → 按需弹小 modal).
     在线 tab: 占位。 -->
<template>
  <div class="asset-panel relative flex h-full min-h-0 flex-col" :data-workspace="workspace">
    <div v-if="drillIn && detailId && !workspace" class="flex h-full min-h-0 flex-col">
      <div class="flex shrink-0 items-center gap-2 border-b border-default px-3 py-2">
        <UButton
          size="xs"
          variant="ghost"
          color="neutral"
          icon="i-tabler-arrow-left"
          @click="drillIn = false"
        >
          {{ t('common.back') }}
        </UButton>
        <span class="min-w-0 flex-1 truncate text-sm font-medium text-highlighted">{{
          lib.byId(detailId)?.label || detailId
        }}</span>
        <UButton size="xs" color="primary" icon="i-tabler-package-import" @click="onDetailInsert">
          {{ t('library.explorer.insert') }}
        </UButton>
      </div>
      <LibraryDetailPanel :sgID="detailId" @insert="onDetailInsert" />
    </div>

    <template v-else>
      <div
        class="flex shrink-0 items-center justify-between gap-3 border-b border-default px-3 py-2.5"
      >
        <div class="min-w-0">
          <p class="text-sm font-semibold text-highlighted">
            {{ t('assetBrowser.automationBlueprints') }}
          </p>
          <p class="text-xs text-dimmed">{{ t('assetBrowser.blueprintSubtitle') }}</p>
        </div>
        <UIcon name="i-tabler-hierarchy" class="size-4 shrink-0 text-primary" />
      </div>

      <div class="flex min-h-0 flex-1 overflow-hidden">
        <AssetCategoryRail
          v-if="workspace"
          v-model="categoryFilter"
          :items="categoryFilterItems"
          class="asset-category-rail"
        />
        <div class="flex min-w-0 flex-1 flex-col gap-2.5 overflow-hidden p-3">
          <AssetBrowserToolbar
            ref="toolbarRef"
            v-model:query="query"
            v-model:category="categoryFilter"
            v-model:tags="tagFilter"
            v-model:sort-key="sortKey"
            v-model:sort-desc="sortDesc"
            v-model:view="viewMode"
            :search-placeholder="t('library.explorer.search')"
            :category-items="categoryFilterItems"
            :tag-items="allTags"
            :sort-items="sortItems"
            allow-view-switch
            :show-search="!workspace"
            :show-category-scopes="!workspace"
          />

          <div class="flex shrink-0 items-center justify-between gap-3 px-0.5">
            <p class="text-xs text-dimmed">{{ t('editor.dock.drag_hint') }}</p>
            <span class="text-xs text-dimmed">{{
              t('library.toolbar.total', { n: pageResult.total })
            }}</span>
          </div>
          <AssetSelectionBar
            :count="selected.size"
            :batch-items="batchMenuItems"
            @clear="selClear()"
          />

          <div data-asset-browser-list class="min-h-0 flex-1 overflow-y-auto pr-1 select-none">
            <div
              v-if="filteredItems.length === 0"
              class="flex min-h-56 flex-col items-center justify-center text-center"
            >
              <div class="mb-3 flex size-11 items-center justify-center rounded-lg bg-elevated/60">
                <UIcon
                  :name="lib.loading ? 'i-tabler-loader-2' : 'i-tabler-hierarchy-off'"
                  class="size-5 text-muted"
                  :class="{ 'animate-spin': lib.loading }"
                />
              </div>
              <p class="text-sm font-medium text-toned">
                <span v-if="lib.loading">{{ t('library.loading') }}</span>
                <span v-else-if="lib.subgraphs.length === 0">{{
                  t('library.explorer.empty')
                }}</span>
                <span v-else>{{ t('library.explorer.no_match') }}</span>
              </p>
            </div>

            <div v-else class="space-y-1">
              <template v-for="group in groupedItems" :key="group.category">
                <div class="px-1 pb-1.5 pt-3 text-xs font-medium text-muted">
                  {{ group.category }}
                </div>
                <div
                  role="listbox"
                  :aria-label="group.category"
                  aria-multiselectable="true"
                  data-asset-list
                  class="blueprint-grid grid gap-3"
                  :class="viewMode === 'list' ? 'blueprint-grid--list' : ''"
                >
                  <UContextMenu
                    v-for="item in group.items"
                    :key="item.id"
                    :items="ctxMenuItems(item)"
                  >
                    <div
                      draggable="true"
                      role="option"
                      data-asset-option
                      :data-asset-id="item.id"
                      :tabindex="isTabStop(item.id) ? 0 : -1"
                      :aria-selected="isSelected(item.id)"
                      :aria-label="t('editor.dock.select_asset', { name: item.label })"
                      class="group cursor-grab overflow-hidden rounded-lg border bg-default active:cursor-grabbing focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                      :class="
                        isSelected(item.id)
                          ? 'border-primary bg-primary/5'
                          : 'border-default hover:border-accented hover:bg-elevated/25'
                      "
                      @click="onRowClick(item.id, $event)"
                      @dblclick="onPick(item.id)"
                      @focus="setActive(item.id)"
                      @keydown="onRowKeydown(item.id, $event)"
                      @contextmenu="selClick(item.id)"
                      @dragstart="
                        (e) => startEditorDrag({ type: 'library-subgraph', id: item.id }, e)
                      "
                    >
                      <div
                        class="blueprint-preview aspect-[15/7] border-b border-default bg-sunken"
                      >
                        <BlueprintTopologyPreview :graph="item.graph" />
                      </div>
                      <div class="min-w-0 p-3">
                        <div class="flex items-start gap-2">
                          <div class="min-w-0 flex-1">
                            <div class="flex items-center gap-2">
                              <span class="truncate text-[13px] font-semibold text-highlighted">{{
                                item.label
                              }}</span>
                              <UIcon
                                v-if="isSelected(item.id)"
                                name="i-tabler-circle-check-filled"
                                class="size-4 shrink-0 text-primary"
                              />
                            </div>
                            <p
                              v-if="item.description"
                              class="mt-1 line-clamp-2 text-xs leading-relaxed text-dimmed"
                            >
                              {{ item.description }}
                            </p>
                          </div>
                          <UButton
                            size="xs"
                            variant="ghost"
                            color="neutral"
                            icon="i-tabler-dots"
                            class="size-7 shrink-0 p-0 opacity-0 group-hover:opacity-100 group-focus-within:opacity-100"
                            :aria-label="t('editor.dock.detail')"
                            @click.stop="openDetail(item.id)"
                            @dblclick.stop
                          />
                        </div>
                        <div
                          class="mt-2 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-dimmed"
                        >
                          <span>{{
                            t('assetBrowser.nodeCount', { n: item.graph?.nodes?.length ?? 0 })
                          }}</span>
                          <span>{{
                            t('assetBrowser.outputCount', { n: item.outputPins?.length ?? 0 })
                          }}</span>
                          <span v-if="item.requiredGlobals?.length">{{
                            t('assetBrowser.requiredVariables', { n: item.requiredGlobals.length })
                          }}</span>
                        </div>
                      </div>
                    </div>
                  </UContextMenu>
                </div>
              </template>
            </div>
          </div>

          <AssetPager
            v-model:page="page"
            :total="pageResult.total"
            :total-pages="pageResult.totalPages"
            :items-per-page="pageSize"
          />
        </div>

        <AssetWorkspaceInspector
          v-if="workspace"
          :open="drillIn && !!detailId"
          :title="
            (detailId && (lib.subgraphs.find((item) => item.id === detailId)?.label || detailId)) ||
            t('assetBrowser.automationBlueprints')
          "
          @close="drillIn = false"
        >
          <LibraryDetailPanel :sgID="detailId" @insert="onDetailInsert" />
        </AssetWorkspaceInspector>
      </div>
    </template>
  </div>

  <!-- 批量加标签 -->
  <BaseModal
    v-model:open="batchTagsOpen"
    :title="t('library.batch.add_tags_title')"
    icon="i-tabler-tags"
    size="md"
  >
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
      <UButton variant="ghost" color="neutral" @click="batchTagsOpen = false">{{
        t('common.cancel')
      }}</UButton>
      <UButton color="primary" :disabled="batchTags.length === 0" @click="onBatchAddTags">{{
        t('library.batch.add_tags_apply')
      }}</UButton>
    </template>
  </BaseModal>

  <!-- 批量改分类 -->
  <BaseModal
    v-model:open="batchCategoryOpen"
    :title="t('library.batch.change_category_title')"
    icon="i-tabler-category"
    size="md"
  >
    <UInputMenu
      v-model="batchCategory"
      :create-item="'always'"
      :items="allCategories"
      size="sm"
      :placeholder="t('library.batch.change_category_placeholder')"
      @create="(v: string) => (batchCategory = v)"
    />
    <template #footer>
      <UButton variant="ghost" color="neutral" @click="batchCategoryOpen = false">{{
        t('common.cancel')
      }}</UButton>
      <UButton color="primary" @click="onBatchChangeCategory">{{
        t('library.batch.change_category_apply')
      }}</UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch, nextTick, useTemplateRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { useLocalStorage } from '@vueuse/core'
import { useLibraryStore } from '@/stores/library'
import { useConfirm } from '@/composables/useConfirm'
import { useListSelection } from '@/composables/editor/useListSelection'
import { filterSubgraphs, groupByCategory, paginate } from '@/lib/libraryFilter'
import { startEditorDrag } from '@/composables/editor/useEditorDragDrop'
import BaseModal from '@/components/common/BaseModal.vue'
import AssetSelectionBar from './AssetSelectionBar.vue'
import AssetBrowserToolbar from './AssetBrowserToolbar.vue'
import AssetPager from './AssetPager.vue'
import AssetCategoryRail from './AssetCategoryRail.vue'
import AssetWorkspaceInspector from './AssetWorkspaceInspector.vue'
import BlueprintTopologyPreview from './BlueprintTopologyPreview.vue'
import LibraryDetailPanel from '@/components/containers/LibraryDetailPanel.vue'
import { backend, type Subgraph } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'
import { useRovingAssetList } from '@/composables/editor/useRovingAssetList'
import { useAssetBrowserPreferences } from '@/composables/editor/useAssetBrowserPreferences'

const { t } = useI18n()
const { workspace = false, workspaceQuery = '' } = defineProps<{
  workspace?: boolean
  workspaceQuery?: string
}>()

const emit = defineEmits<{
  'pick-subgraph': [libraryID: string]
}>()

const { query, categoryFilter, tagFilter, sortKey, sortDesc, viewMode } =
  useAssetBrowserPreferences<'label' | 'createdAt' | 'nodes'>('blueprints', 'label')
const effectiveQuery = computed(() => workspaceQuery.trim() || query.value)
const toolbarRef = useTemplateRef<{ focusSearch: () => Promise<void> }>('toolbarRef')

// 排序 (镜像模板/clip 管理): 名称/创建时间/节点数 × 正逆序.
const sortItems = computed(() => [
  { label: t('library.explorer.view_by_name'), value: 'label' },
  { label: t('library.explorer.view_by_created'), value: 'createdAt' },
  { label: t('library.explorer.view_by_nodes'), value: 'nodes' },
])

const lib = useLibraryStore()
const toast = useToast()
const { confirm } = useConfirm()

// ── 过滤 + 分组 ──
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
    query: effectiveQuery.value,
    category:
      categoryFilter.value === 'all'
        ? null
        : categoryFilter.value === 'none'
          ? ''
          : categoryFilter.value.slice(2),
    tags: tagFilter.value,
  })
  const sorted = [...arr]
  sorted.sort((a, b) => {
    let cmp = 0
    switch (sortKey.value) {
      case 'label':
        cmp = (a.label ?? '').localeCompare(b.label ?? '')
        break
      case 'createdAt':
        cmp = (a.createdAt ?? '').localeCompare(b.createdAt ?? '')
        break
      case 'nodes':
        cmp = (a.graph?.nodes?.length ?? 0) - (b.graph?.nodes?.length ?? 0)
        break
    }
    return sortDesc.value ? -cmp : cmp
  })
  return sorted
})

const page = ref(1)
const pageSize = useLocalStorage('library.pageSize', 50)
const pageResult = computed(() => paginate(filteredItems.value, page.value, pageSize.value))
const groupedItems = computed(() =>
  groupByCategory(pageResult.value.pageItems, t('library.explorer.uncategorized')),
)

watch([effectiveQuery, categoryFilter, tagFilter, pageSize, sortKey, sortDesc], () => {
  page.value = 1
})
watch(
  () => pageResult.value.totalPages,
  (tp) => {
    if (page.value > tp) page.value = tp
  },
)

// 选中 (单击/Ctrl/Shift/勾选框) — 用于批量操作.
const visibleIds = computed(() => groupedItems.value.flatMap((g) => g.items.map((i) => i.id)))
const { selected, click: selClick, clear: selClear, isSelected } = useListSelection(visibleIds)
const { isTabStop, setActive, move } = useRovingAssetList(visibleIds)

// 数据由 AssetDockPanel 统一预载；浏览偏好保留到下次打开。
onMounted(async () => {
  await nextTick()
  await toolbarRef.value?.focusSearch()
})

function onRowClick(id: string, e: MouseEvent) {
  detailId.value = id
  if (workspace && !e.ctrlKey && !e.metaKey && !e.shiftKey) drillIn.value = true
  selClick(id, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey })
}

function onRowKeydown(id: string, e: KeyboardEvent) {
  if (e.target !== e.currentTarget) return
  if (move(id, e)) return
  if (e.key === 'Enter') {
    e.preventDefault()
    onPick(id)
    return
  }
  if (e.key !== ' ') return
  e.preventDefault()
  selClick(id)
}

function onPick(libraryID: string) {
  emit('pick-subgraph', libraryID)
}

// ── 详情 (按需弹出) ──
const detailId = ref<string | null>(null)
const drillIn = ref(false)
function openDetail(id: string) {
  detailId.value = id
  drillIn.value = true
}
function onDetailInsert() {
  if (detailId.value) onPick(detailId.value)
  if (!workspace) drillIn.value = false
}

function ctxMenuItems(item: Subgraph) {
  return [
    [
      {
        label: t('library.explorer.insert'),
        icon: 'i-tabler-package-import',
        onSelect: () => onPick(item.id),
      },
      {
        label: t('editor.dock.detail'),
        icon: 'i-tabler-info-circle',
        onSelect: () => openDetail(item.id),
      },
      {
        label: t('library.card.duplicate'),
        icon: 'i-tabler-copy-plus',
        onSelect: () => onDuplicate(item),
      },
    ],
    [{ label: t('library.card.copy_id'), icon: 'i-tabler-copy', onSelect: () => onCopyID(item) }],
    [
      {
        label: t('library.card.delete'),
        icon: 'i-tabler-trash',
        color: 'error' as const,
        onSelect: () => onDelete(item),
      },
    ],
  ]
}

async function onCopyID(item: Subgraph) {
  try {
    await navigator.clipboard.writeText(item.id)
    toast.add({
      title: t('toast.copy_id_success'),
      color: 'success',
      icon: 'i-tabler-check',
      duration: 1500,
    })
  } catch (e: any) {
    toast.add({ title: t('toast.copy_failed'), description: errorMessage(e), color: 'error' })
  }
}

// 复制为新子图 (fork): 想独立改不影响引用方时用。
async function onDuplicate(item: Subgraph) {
  const dup = await lib.duplicateSubgraph(item.id)
  if (dup) {
    toast.add({
      title: t('library.card.duplicated', { name: dup.label }),
      color: 'success',
      icon: 'i-tabler-check',
    })
  }
}

// 删除安全: 先扫引用 — 被容器使用时警告里带"被 N 个容器使用", 确认才真删。
async function onDelete(item: Subgraph) {
  const refs = await lib.referrersOf(item.id)
  const useCount = lib.containerUseCount(refs)
  const name = item.label || item.id
  const desc =
    useCount > 0
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
  const desc =
    referenced.length > 0
      ? t('library.batch.delete_confirm_referenced', {
          n: ids.length,
          m: referenced.length,
          names: referenced.join('、'),
        })
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
    if (!sg) {
      failed++
      continue
    }
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
    {
      label: t('library.batch.add_tags'),
      icon: 'i-tabler-tags',
      onSelect: () => {
        batchTagsOpen.value = true
      },
    },
    {
      label: t('library.batch.change_category'),
      icon: 'i-tabler-category',
      onSelect: () => {
        batchCategoryOpen.value = true
      },
    },
  ],
  [
    {
      label: t('library.batch.delete'),
      icon: 'i-tabler-trash',
      color: 'error' as const,
      onSelect: () => {
        void onBatchDelete()
      },
    },
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
    if (!sg) {
      failed++
      continue
    }
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

<style scoped>
.asset-panel {
  container-type: inline-size;
}

.blueprint-grid {
  grid-template-columns: repeat(auto-fill, minmax(210px, 1fr));
}

.blueprint-grid--list {
  grid-template-columns: 1fr;
}

.blueprint-grid--list [data-asset-option] {
  display: grid;
  grid-template-columns: 150px minmax(0, 1fr);
}

.blueprint-grid--list .blueprint-preview {
  border-bottom: 0;
  border-right: 1px solid var(--ui-border);
}

@container (width < 760px) {
  [data-workspace='true'] .asset-category-rail {
    display: none;
  }
}

@container (width < 520px) {
  .blueprint-grid {
    grid-template-columns: 1fr;
  }
}
</style>
